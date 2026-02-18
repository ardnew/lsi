// Package main implements lsi, a command-line utility for analyzing file paths.
//
// lsi interprets its arguments as pathnames to any type of Unix file (symlinks,
// files, directories, and so forth). It then follows each pathname until an
// endpoint is found (a file, a directory, a device node, etc). If it finds a
// symbolic link, it shows the link, and starts following it, indenting the
// output to show the context.
//
// This tool provides the same functionality as the namei utility from the
// util-linux package.
package main
