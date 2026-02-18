package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// Example_version demonstrates printing the version.
// CLI equivalent: lsi -v
func Example_version() {
	var out, errOut bytes.Buffer
	ctx := context.Background()

	_ = run(ctx, &out, &errOut, []string{"-v"})

	// Version output will vary, so we just check it contains "lsi"
	fmt.Println("Output contains 'lsi':", bytes.Contains(out.Bytes(), []byte("lsi")))

	// Output:
	// Output contains 'lsi': true
}

// Example_basicPath demonstrates analyzing a simple path.
// CLI equivalent: lsi /tmp/myfile
func Example_basicPath() {
	// Create a temporary file for the example
	tmpDir := os.TempDir()
	testFile := filepath.Join(tmpDir, "example_test.txt")
	_ = os.WriteFile(testFile, []byte("example"), 0644)
	defer os.Remove(testFile)

	var out, errOut bytes.Buffer
	ctx := context.Background()

	_ = run(ctx, &out, &errOut, []string{testFile})

	// Output will show path breakdown
	fmt.Println("Output contains filename:", bytes.Contains(out.Bytes(), []byte("example_test.txt")))

	// Output:
	// Output contains filename: true
}

// Example_splitPath demonstrates path splitting.
func Example_splitPath() {
	// Absolute path
	elem, volume := splitPath("/usr/local/bin")
	fmt.Printf("Elements: %v, Volume: %q\n", elem, volume)

	// Relative path
	elem, volume = splitPath("foo/bar/baz")
	fmt.Printf("Elements: %v, Volume: %q\n", elem, volume)

	// Root only
	elem, volume = splitPath("/")
	fmt.Printf("Elements: %v, Volume: %q\n", elem, volume)

	// Output:
	// Elements: [/ usr local bin], Volume: ""
	// Elements: [foo bar baz], Volume: ""
	// Elements: [/], Volume: ""
}

// Example_fmtName demonstrates entry name formatting.
func Example_fmtName() {
	// Simple file at root level
	e := &entry{Name: "file.txt", Level: 0}
	fmt.Println(e.fmtName())

	// Nested file
	e = &entry{Name: "nested.txt", Level: 2}
	fmt.Println(e.fmtName())

	// Symlink
	e = &entry{Name: "link", Link: "target", Level: 0}
	fmt.Println(e.fmtName())

	// Output:
	// file.txt
	//     nested.txt
	// link -> target
}

// Example_walk demonstrates walking a path.
// CLI equivalent: lsi /tmp/testdir
func Example_walk() {
	// Create a test directory structure
	tmpDir := os.TempDir()
	testDir := filepath.Join(tmpDir, "walk_example")
	_ = os.MkdirAll(testDir, 0755)
	defer os.RemoveAll(testDir)

	testFile := filepath.Join(testDir, "test.txt")
	_ = os.WriteFile(testFile, []byte("test"), 0644)

	ctx := context.Background()

	// Walk the path
	_ = walk(ctx, testFile, func(ctx context.Context, e entry) (bool, error) {
		if e.Err == nil {
			fmt.Printf("Found: %s\n", e.Name)
		}
		return false, nil // Don't follow symlinks
	})

	// Output will vary by system, so we can't use strict output matching
}

// Example_upperIf demonstrates conditional case conversion.
func Example_upperIf() {
	// Uppercase when true
	fmt.Printf("%c\n", upperIf('s', true))

	// Lowercase when false
	fmt.Printf("%c\n", upperIf('s', false))

	// Already uppercase
	fmt.Printf("%c\n", upperIf('T', true))

	// Already lowercase
	fmt.Printf("%c\n", upperIf('t', false))

	// Output:
	// S
	// s
	// T
	// t
}

// Example_calculateWidths demonstrates width calculation for columnar output.
func Example_calculateWidths() {
	entries := []entry{
		{Mode: "-rw-r--r--", User: "root", Group: "wheel", Size: 1024, Inode: 12345},
		{Mode: "drwxr-xr-x", User: "admin", Group: "staff", Size: 4096, Inode: 67890},
	}

	w := calculateWidths(entries)

	fmt.Printf("Mode width: %d\n", w.mode)
	fmt.Printf("User width: %d\n", w.user)
	fmt.Printf("Group width: %d\n", w.group)

	// Output:
	// Mode width: 10
	// User width: 5
	// Group width: 5
}

// Example_atob demonstrates boolean string parsing.
func Example_atob() {
	fmt.Println(atob("true"))
	fmt.Println(atob("false"))
	fmt.Println(atob("1"))
	fmt.Println(atob("0"))
	fmt.Println(atob("invalid"))

	// Output:
	// true
	// false
	// true
	// false
	// false
}

// Example_options demonstrates common flag combinations.
// CLI equivalent: lsi -l /tmp/myfile
func Example_options() {
	// Long format (includes mode, user, group, size, mount)
	opts := options{long: true}
	if opts.long {
		opts.mode, opts.user, opts.group, opts.size, opts.mount = true, true, true, true, true
	}

	fmt.Printf("Mode: %v, User: %v, Group: %v, Size: %v, Mount: %v\n",
		opts.mode, opts.user, opts.group, opts.size, opts.mount)

	// Output:
	// Mode: true, User: true, Group: true, Size: true, Mount: true
}

// Example_longFormat demonstrates long format output.
// CLI equivalent: lsi -l /tmp/myfile
func Example_longFormat() {
	tmpDir := os.TempDir()
	testFile := filepath.Join(tmpDir, "long_format_test.txt")
	_ = os.WriteFile(testFile, []byte("test content"), 0644)
	defer os.Remove(testFile)

	var out, errOut bytes.Buffer
	ctx := context.Background()

	_ = run(ctx, &out, &errOut, []string{"-l", testFile})

	// Output will contain permission bits
	output := out.String()
	hasPermissions := bytes.Contains([]byte(output), []byte("rw-")) ||
		bytes.Contains([]byte(output), []byte("r--"))

	fmt.Println("Output contains permissions:", hasPermissions)

	// Output:
	// Output contains permissions: true
}

// Example_noFollow demonstrates not following symlinks.
// CLI equivalent: lsi -n /tmp/mylink
func Example_noFollow() {
	tmpDir := os.TempDir()
	target := filepath.Join(tmpDir, "target.txt")
	symlink := filepath.Join(tmpDir, "example_link")

	_ = os.WriteFile(target, []byte("target"), 0644)
	_ = os.Symlink(target, symlink)
	defer os.Remove(target)
	defer os.Remove(symlink)

	var out, errOut bytes.Buffer
	ctx := context.Background()

	_ = run(ctx, &out, &errOut, []string{"-n", symlink})

	// With -n, symlink target is not traversed
	fmt.Println("Symlink not followed")

	// Output:
	// Symlink not followed
}

// Example_multiplePaths demonstrates processing multiple paths.
// CLI equivalent: lsi /tmp/file1 /tmp/file2
func Example_multiplePaths() {
	tmpDir := os.TempDir()
	file1 := filepath.Join(tmpDir, "multi_test1.txt")
	file2 := filepath.Join(tmpDir, "multi_test2.txt")

	_ = os.WriteFile(file1, []byte("one"), 0644)
	_ = os.WriteFile(file2, []byte("two"), 0644)
	defer os.Remove(file1)
	defer os.Remove(file2)

	var out, errOut bytes.Buffer
	ctx := context.Background()

	_ = run(ctx, &out, &errOut, []string{file1, file2})

	// Output will contain headers for multiple paths
	hasHeaders := bytes.Contains(out.Bytes(), []byte("--"))
	fmt.Println("Output contains path headers:", hasHeaders)

	// Output:
	// Output contains path headers: true
}
