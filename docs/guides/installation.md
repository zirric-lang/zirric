---
title: Installation
description: How to install Zirric on your system
---

# Installation

This guide covers available distribution packages. Zirric releases are still experimental, so expect alpha versions and occasional packaging changes.

::: callout warning Experimental packages
These packages may lag behind main and may change their version scheme.
:::

## Linux

### Arch (btw)

Package index: https://code.knabel.dev/zirric-lang/-/packages/arch/zirric

1. Add the repository signing key:

```bash
wget -O sign.gpg https://code.knabel.dev/api/packages/zirric-lang/arch/repository.key
sudo pacman-key --add sign.gpg
sudo pacman-key --lsign-key 'zirric-lang@noreply.code.knabel.dev'
```

2. Add the repository to `/etc/pacman.conf`:

```toml
[zirric-lang.code.knabel.dev]
SigLevel = Required
Server = https://code.knabel.dev/api/packages/zirric-lang/arch/extras/$arch
```

3. Install:

```bash
sudo pacman -Sy zirric
```

For more details, see the Forgejo Arch registry documentation: https://forgejo.org/docs/latest/user/packages/arch/

### Alpine

Package index: https://code.knabel.dev/zirric-lang/-/packages/alpine/zirric

1. Add the registry URL to `/etc/apk/repositories`:

```
https://code.knabel.dev/api/packages/zirric-lang/alpine/$branch/$repository
```

2. Add the registry key:

```bash
sudo curl -JO https://code.knabel.dev/api/packages/zirric-lang/alpine/key
sudo mv key /etc/apk/keys/zirric-lang.rsa.pub
```

3. Install the package:

```bash
sudo apk add zirric
```

For more details, see the Forgejo Alpine registry documentation: https://forgejo.org/docs/latest/user/packages/alpine/

### Debian

Package index: https://code.knabel.dev/zirric-lang/-/packages/debian/zirric

1. Add the repository signing key and source:

```bash
sudo curl https://code.knabel.dev/api/packages/zirric-lang/debian/repository.key \
  -o /etc/apt/keyrings/forgejo-zirric-lang.asc

echo "deb [signed-by=/etc/apt/keyrings/forgejo-zirric-lang.asc] https://code.knabel.dev/api/packages/zirric-lang/debian $distribution $component" \
  | sudo tee -a /etc/apt/sources.list.d/forgejo.list
```

2. Update and install:

```bash
sudo apt update
sudo apt install zirric
```

For more details, see the Forgejo Debian registry documentation: https://forgejo.org/docs/latest/user/packages/debian/

### Linuxbrew

Install using [Homebrew on Linux](https://docs.brew.sh/Homebrew-on-Linux):

```bash
brew tap vknabel/zirric https://code.knabel.dev/zirric-lang/homebrew-tap.git
brew install zirric
```

## macOS

### Homebrew

Homebrew packages support Linux and macOS on Intel and ARM.

```bash
brew tap vknabel/zirric https://code.knabel.dev/zirric-lang/homebrew-tap.git
brew install zirric
```

## Cross-Platform

### asdf

Install the asdf plugin from [asdf-zirric](https://code.knabel.dev/zirric-lang/asdf-zirric.git):

```bash
asdf plugin add zirric https://code.knabel.dev/zirric-lang/asdf-zirric.git

# Show all installable versions
asdf list-all zirric

# Install specific version
asdf install zirric latest

# Set a version globally (on your ~/.tool-versions file)
asdf global zirric latest

# Now zirric commands are available
zirric --help
```

### Docker

Pull the latest container image:

```bash
docker pull code.knabel.dev/zirric-lang/zirric:latest
```
