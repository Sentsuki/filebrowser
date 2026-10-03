# Installation

File Browser is a single binary and can be used as standalone executable. The installation and first time setup is quite straightforward independently of which system you use.

## Binary

The quickest and easiest way to install File Browser is to use a package manager, or our download script, which automatically fetches the latest version of File Browser for your platform. Alternatively, you can manually download the binary from the [releases page](https://github.com/filebrowser/filebrowser/releases).

### Brew

```sh
brew tap filebrowser/tap
brew install filebrowser
filebrowser -r /path/to/your/files
```

### Unix

```sh
curl -fsSL https://raw.githubusercontent.com/filebrowser/get/master/get.sh | bash
filebrowser -r /path/to/your/files
```

### Windows

```sh
iwr -useb https://raw.githubusercontent.com/filebrowser/get/master/get.ps1 | iex
filebrowser -r /path/to/your/files
```

File Browser is now up and running. Read the ["First Boot"](#first-boot) section for more information.

## First Boot

Your instance is now up and running. File Browser will automatically bootstrap a database, in which the configuration and the users are stored. You can find the address in which your instance is running, as well as the randomly generated password for the user `admin`, in the console logs.

> [!WARNING]
>
> The automatically generated password for the user `admin` is only displayed once. If you fail to remember it, you will need to manually delete the database and start File Browser again.

Although this is the fastest way to bootstrap an instance, we recommend you to take a look at other possible options, by checking [`config init`](cli/filebrowser-config-init.md) and [`config set`](cli/filebrowser-config-set.md), to make the installation as safe and customized as it can be.

If your goal is to have a public-facing deployment, we recommend taking a look at the [deployment](deployment.md) page for more information on how you can secure your installation.
