package fbhttp

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/asdine/storm/v3"
	"github.com/spf13/afero"

	"github.com/filebrowser/filebrowser/v2/auth"
	"github.com/filebrowser/filebrowser/v2/files"
	"github.com/filebrowser/filebrowser/v2/settings"
	"github.com/filebrowser/filebrowser/v2/share"
	"github.com/filebrowser/filebrowser/v2/storage/bolt"
	"github.com/filebrowser/filebrowser/v2/users"
)

// publicRoutesHandler builds the full router with a public directory share "h"
// rooted at "/shared" (containing "a.txt") and a minimal asset filesystem whose
// index page renders only the injected StaticURL.
func publicRoutesHandler(t *testing.T, baseURL string) http.Handler {
	t.Helper()

	db, err := storm.Open(filepath.Join(t.TempDir(), "db"))
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	st, err := bolt.NewStorage(db)
	if err != nil {
		t.Fatalf("failed to get storage: %v", err)
	}
	if err := st.Share.Save(&share.Link{Hash: "h", UserID: 1, Path: "/shared"}); err != nil {
		t.Fatalf("failed to save share: %v", err)
	}
	if err := st.Users.Save(&users.User{
		Username: "username",
		Password: "pw",
		Perm:     users.Permissions{Share: true, Download: true},
	}); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}
	if err := st.Settings.Save(&settings.Settings{Key: []byte("key"), AuthMethod: auth.MethodJSONAuth}); err != nil {
		t.Fatalf("failed to save settings: %v", err)
	}
	if err := st.Auth.Save(&auth.JSONAuth{}); err != nil {
		t.Fatalf("failed to save auther: %v", err)
	}

	scope := t.TempDir()
	if err := os.MkdirAll(filepath.Join(scope, "shared"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(scope, "shared", "a.txt"), []byte("shared-content"), 0o644); err != nil {
		t.Fatal(err)
	}
	st.Users = &customFSUser{Store: st.Users, fs: files.NewScopedFs(afero.NewOsFs(), scope)}

	assets := fstest.MapFS{
		"public/index.html": {Data: []byte("index:[{[ .StaticURL ]}]")},
		"app.css":           {Data: []byte("css-content")},
	}

	handler, err := NewHandler(nil, nil, nil, st, &settings.Server{Root: t.TempDir(), BaseURL: baseURL}, assets)
	if err != nil {
		t.Fatalf("failed to build handler: %v", err)
	}
	return handler
}

func getPath(t *testing.T, h http.Handler, target string) (int, string, http.Header) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, http.NoBody))
	res := rec.Result()
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}
	return res.StatusCode, string(body), res.Header
}

// Everything an anonymous share recipient needs is served under "/share/", so
// an access proxy can exempt that single prefix.
func TestPublicShareRoutesLiveUnderSharePrefix(t *testing.T) {
	t.Parallel()

	for _, baseURL := range []string{"", "/fb"} {
		t.Run("baseURL="+baseURL, func(t *testing.T) {
			t.Parallel()
			h := publicRoutesHandler(t, baseURL)

			cases := []struct {
				path, want string
			}{
				{"/share/h", "index:" + baseURL + "/share/_/static"},
				{"/share/_/static/app.css", "css-content"},
				{"/share/_/api/share/h", `"name":"a.txt"`},
				{"/share/_/api/dl/h/a.txt", "shared-content"},
			}
			for _, tc := range cases {
				status, body, _ := getPath(t, h, baseURL+tc.path)
				if status != http.StatusOK || !strings.Contains(body, tc.want) {
					t.Errorf("GET %s: got %d %q, want 200 containing %q", baseURL+tc.path, status, body, tc.want)
				}
			}
		})
	}
}

// The previous public locations are gone: they fall through to the index page
// instead of serving share content or assets.
func TestPublicShareLegacyRoutesRemoved(t *testing.T) {
	t.Parallel()
	h := publicRoutesHandler(t, "")

	for _, path := range []string{
		"/api/public/share/h",
		"/api/public/dl/h/a.txt",
		"/static/app.css",
	} {
		status, body, _ := getPath(t, h, path)
		if status != http.StatusOK || body != "index:/share/_/static" {
			t.Errorf("GET %s: got %d %q, want the index page", path, status, body)
		}
	}
}

// A path that lexically climbs out of "/share/" is not served from inside the
// prefix; it is redirected to its clean form, which the proxy then evaluates.
func TestPublicSharePrefixTraversalRedirects(t *testing.T) {
	t.Parallel()
	h := publicRoutesHandler(t, "")

	status, _, header := getPath(t, h, "/share/_/api/dl/../../../../api/resources/")
	if status != http.StatusMovedPermanently || header.Get("Location") != "/api/resources/" {
		t.Errorf("got %d Location=%q, want 301 to /api/resources/", status, header.Get("Location"))
	}
}
