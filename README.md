[docimg]:https://godoc.org/github.com/ardnew/lsi?status.svg
[docurl]:https://godoc.org/github.com/ardnew/lsi
[repimg]:https://goreportcard.com/badge/github.com/ardnew/lsi
[repurl]:https://goreportcard.com/report/github.com/ardnew/lsi

# lsi
#### lsi

[![GoDoc][docimg]][docurl] [![Go Report Card][repimg]][repurl]

## Usage

Running `lsi` without any flags or arguments will simply print the path components from your current directory, one per line:

```sh
$ lsi
/
home
andrew
```

To view a long listing for each of the path components, similar to `ls -l` from GNU Coreutils, use the `-l` flag:

```sh
$ lsi -l
drwxr-xr-x   root   root 4096 @ /
drwxr-xr-x   root   root 4096   home
drwxr-xr-x andrew andrew 4096   andrew
```

Notice that the `-l` flag also indicates whether an individual component represents a mount point using the `@` symbol preceding the file name. Therefore, in the above example, we can be confident all of these files exist on the same physical device.

By default, symlinks encountered are followed up until the two paths coincide, and each level of indirection is represented by indentation preceding the file name. Multiple paths may be specified at once:

```sh
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

Use the `-h` flag for a summary of all options available:

```
usage:
  lsi [flags] [--] [PATH ...]

flags:
  -v  Display version information.
  -a  Display change history.
  -n  Do not follow symlinks.
  -l  Output using long format (-p -u -g -s -m).
  -p  Output file type and permissions.
  -u  Output file owner.
  -g  Output file group.
  -s  Output file size (bytes).
  -i  Output file inode.
  -m  Output mount point symbols (@).
```

## Installation

Use the builtin Go package manager:

```sh
go get -v github.com/ardnew/lsi/cmd/lsi
```
