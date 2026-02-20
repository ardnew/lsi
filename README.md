[docimg]:https://godoc.org/github.com/ardnew/lsi?status.svg
[docurl]:https://godoc.org/github.com/ardnew/lsi
[repimg]:https://goreportcard.com/badge/github.com/ardnew/lsi
[repurl]:https://goreportcard.com/report/github.com/ardnew/lsi

# lsi
#### Follow elements of a file path

[![GoDoc][docimg]][docurl] [![Go Report Card][repimg]][repurl]

`lsi` is a command-line utility for analyzing file paths. It provides similar functionality to the `namei` utility from the [`util-linux` package](https://www.kernel.org/pub/linux/utils/util-linux/):
> __lsi__ interprets its arguments as pathnames to any type of Unix file (symlinks, files, directories, and so forth). __lsi__ then follows each path‐name until an endpoint is found (a file, a directory, a device node, etc). If it finds a symbolic link, it shows the link, and starts following it, indenting the output to show the context.

## Usage

Running `lsi` without any flags or arguments will simply print the path components from your current directory, one per line:

```
$ lsi
/
home
andrew
```

### Flags

`lsi` supports both short (`-flag`) and long (`--flag`) format flags following GNU conventions. Use the `-h` or `--help` flag for a summary of all options:

```
$ lsi --help
lsi - Analyze file paths by traversing and displaying each path component

Usage:
  lsi [flags] [--] [PATH ...]
  lsi completion [SHELL]

Flags:
  -h --help          Display this help message
  -v --version       Display version information
  -t --timeout       Timeout duration (e.g., 30s, 5m)
  -n --no-follow     Do not follow symlinks
  -l --long          Output using long format (-p -u -g -s -m)
  -p --permissions   Output file type and permissions
  -u --user          Output file owner
  -g --group         Output file group
  -s --size          Output file size (bytes)
  -i --inode         Output file inode
  -m --mount         Output mount point symbols (@)

Subcommands:
  completion [SHELL] Generate shell completion script
                     SHELL: bash, zsh, fish, powershell
                     If omitted, auto-detects from environment
```

### Shell Completion

`lsi` can generate shell completion scripts for bash, zsh, fish, and PowerShell. The completion script can either be specified explicitly or auto-detected from your environment:

#### Bash

```sh
# Enable completion for current session
source <(lsi completion bash)

# Add to your ~/.bashrc for persistent completion
echo 'eval "$(lsi completion bash)"' >> ~/.bashrc
```

#### Zsh

```sh
# Enable completion for current session
source <(lsi completion zsh)

# Add to your ~/.zshrc for persistent completion
echo 'eval "$(lsi completion zsh)"' >> ~/.zshrc

# Or save to a file in your $fpath
lsi completion zsh > ~/.zsh/completions/_lsi
```

#### Fish

```sh
# Save to fish completions directory
lsi completion fish > ~/.config/fish/completions/lsi.fish
```

#### PowerShell

```powershell
# Add to your PowerShell profile
lsi completion powershell | Out-String | Invoke-Expression

# Or add this line to your profile:
Invoke-Expression -Command $(lsi completion powershell | Out-String)
```

#### Auto-Detection

If you don't specify a shell, `lsi` will auto-detect your current shell from environment variables:

```sh
# Auto-detects bash, zsh, fish, or powershell
lsi completion
```

### Long Format

To view a long listing for each of the path components, similar to `ls -l` from GNU Coreutils, use the `-l` or `--long` flag:

```
$ lsi -l
drwxr-xr-x   root   root 4096 @ /
drwxr-xr-x   root   root 4096   home
drwxr-xr-x andrew andrew 4096   andrew
```

Notice that the `-l` flag also indicates whether an individual component represents a mount point using the `@` symbol preceding the file name. Therefore, in the above example, we can be confident all of these files exist on the same physical device.

### Symlinks

By default, symlinks encountered are followed up until the two paths coincide, and each level of indirection is represented by indentation preceding the file name. Multiple paths may be specified at once:

```
$  lsi -l /bin/vi /proc/self/exe
-- /bin/vi
drwxr-xr-x root root    4096 @ /
lrwxrwxrwx root root       7   bin -> usr/bin
drwxr-xr-x root root    4096     usr
drwxr-xr-x root root  135168     bin
lrwxrwxrwx root root      20   vi -> /etc/alternatives/vi
drwxr-xr-x root root    4096 @   /
drwxr-xr-x root root   12288     etc
drwxr-xr-x root root   12288     alternatives
lrwxrwxrwx root root      13     vi -> /usr/bin/nvim
drwxr-xr-x root root    4096 @     /
drwxr-xr-x root root    4096       usr
drwxr-xr-x root root  135168       bin
-rwxrwxr-x root root 3469640       nvim

-- /proc/self/exe
drwxr-xr-x   root      root    4096 @ /
dr-xr-xr-x   root      root       0 @ proc
lrwxrwxrwx   root      root       0   self -> 4069059
dr-xr-xr-x andrew    andrew       0     4069059
lrwxrwxrwx andrew    andrew       0   exe -> /usr/local/go/bin/lsi
drwxr-xr-x   root      root    4096 @   /
drwxr-xr-x   root      root    4096     usr
drwxrwsr-x   root developer    4096     local
drwxrwsr-x andrew developer    4096     go
drwxrwsr-x andrew developer    4096     bin
-rwxrwxr-x andrew developer 2871632     lsi
```

Use the `-n` or `--no-follow` flag to prevent following symlinks:

```
$ lsi --no-follow /bin/vi
```

### Timeout Support

The `-t` or `--timeout` flag allows you to set a timeout for path traversal operations, useful when dealing with potentially slow or problematic filesystems:

```sh
# Set a 30 second timeout
$ lsi -t 30s /mnt/slow-network-share

# Using long form with equals syntax
$ lsi --timeout=5m /deep/directory/tree
```

If the timeout is exceeded, `lsi` will report the elapsed time and exit gracefully.

## Installation

### From Releases (Recommended)

Download pre-built binaries from the [releases page](https://github.com/ardnew/lsi/releases):

```sh
# Linux (amd64)
wget https://github.com/ardnew/lsi/releases/latest/download/lsi-VERSION-linux-amd64.tar.gz
tar -xzf lsi-VERSION-linux-amd64.tar.gz
sudo mv lsi-VERSION-linux-amd64/lsi /usr/local/bin/

# macOS (arm64)
wget https://github.com/ardnew/lsi/releases/latest/download/lsi-VERSION-darwin-arm64.tar.gz
tar -xzf lsi-VERSION-darwin-arm64.tar.gz
sudo mv lsi-VERSION-darwin-arm64/lsi /usr/local/bin/

# Windows (amd64)
# Download lsi-VERSION-windows-amd64.zip from releases page
# Extract and add to PATH
```

### From Source

Requires Go 1.21 or later:

```sh
go install github.com/ardnew/lsi@latest
```
