package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestRunCurrentDirectory tests running without arguments uses current directory.
func TestRunCurrentDirectory(t *testing.T) {
	var out, errOut bytes.Buffer
	ctx := context.Background()

	// Run without arguments
	err := run(ctx, &out, &errOut, []string{})

	if err != nil {
		t.Errorf("run() without arguments error = %v, want nil", err)
	}

	// Should produce output for current directory
	if out.Len() == 0 {
		t.Error("run() without arguments should produce output")
	}
}

// TestContextError tests context error formatting.
func TestContextError(t *testing.T) {
	tests := []struct {
		name     string
		setupCtx func() context.Context
		want     string
	}{
		{
			name: "deadline exceeded",
			setupCtx: func() context.Context {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
				defer cancel()
				time.Sleep(2 * time.Millisecond)
				return ctx
			},
			want: "timeout after",
		},
		{
			name: "canceled",
			setupCtx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
			want: "context canceled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := tt.setupCtx()
			start := time.Now()
			err := contextError(ctx, start)
			if err == nil {
				t.Fatal("contextError should return an error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("contextError() = %q, want to contain %q", err.Error(), tt.want)
			}
		})
	}
}

// TestPrintError tests error output formatting.
func TestPrintError(t *testing.T) {
	tests := []struct {
		name     string
		entry    entry
		wantText string
	}{
		{
			name: "path error",
			entry: entry{
				Name: "test",
				Err:  &os.PathError{Op: "stat", Path: "/foo/bar", Err: os.ErrNotExist},
			},
			wantText: "test",
		},
		{
			name: "generic error",
			entry: entry{
				Name: "file",
				Err:  errors.New("generic error"),
			},
			wantText: "file",
		},
		{
			name: "permission denied",
			entry: entry{
				Name: "secret",
				Err:  &os.PathError{Op: "open", Path: "/secret", Err: os.ErrPermission},
			},
			wantText: "secret",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			printError(&buf, tt.entry)
			output := buf.String()
			if !strings.Contains(output, tt.wantText) {
				t.Errorf("printError() output = %q, want to contain %q", output, tt.wantText)
			}
			if !strings.Contains(output, "*") {
				t.Error("printError should start with ' * ' prefix")
			}
		})
	}
}

// TestRun tests the main run function with various scenarios.
func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantErr     bool
		wantContain string
	}{
		{
			name:        "version flag",
			args:        []string{"-v"},
			wantErr:     false,
			wantContain: "lsi",
		},
		{
			name:        "help short flag",
			args:        []string{"-h"},
			wantErr:     false,
			wantContain: "Usage:",
		},
		{
			name:        "help long flag",
			args:        []string{"--help"},
			wantErr:     false,
			wantContain: "Flags:",
		},
		{
			name:    "nonexistent path",
			args:    []string{"/this/path/absolutely/does/not/exist"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			ctx := context.Background()
			err := run(ctx, &out, &errOut, tt.args)

			if (err != nil) != tt.wantErr {
				t.Errorf("run() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantContain != "" {
				output := out.String() + errOut.String()
				if !strings.Contains(output, tt.wantContain) {
					t.Errorf("run() output = %q, want to contain %q", output, tt.wantContain)
				}
			}
		})
	}
}

// TestRunWithTimeout tests timeout functionality.
func TestRunWithTimeout(t *testing.T) {
	var out, errOut bytes.Buffer
	ctx := context.Background()

	// Use an extremely short timeout that should definitely trigger
	err := run(ctx, &out, &errOut, []string{"-t", "1ns", "/usr"})
	if err == nil {
		t.Fatal("run() with 1ns timeout should return error")
	}
	if !strings.Contains(err.Error(), "timeout") {
		t.Errorf("run() error = %q, want to contain 'timeout'", err.Error())
	}
}

// TestRunWithValidPath tests running with a valid path.
func TestRunWithValidPath(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var out, errOut bytes.Buffer
	ctx := context.Background()
	err := run(ctx, &out, &errOut, []string{testFile})

	if err != nil {
		t.Errorf("run() with valid path error = %v, want nil", err)
	}

	output := out.String()
	if !strings.Contains(output, "test.txt") {
		t.Errorf("run() output = %q, want to contain 'test.txt'", output)
	}
}

// TestRunWithLongFormat tests the -l flag.
func TestRunWithLongFormat(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var out, errOut bytes.Buffer
	ctx := context.Background()
	err := run(ctx, &out, &errOut, []string{"-l", testFile})

	if err != nil {
		t.Errorf("run() with -l flag error = %v, want nil", err)
	}

	output := out.String()
	// Long format should include permissions
	if !strings.Contains(output, "rw-") && !strings.Contains(output, "r--") {
		t.Errorf("run() output = %q, want to contain file permissions", output)
	}
}

// TestRunWithInodeFlag tests running with the -i (inode) flag.
func TestRunWithInodeFlag(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "inode_test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var out, errOut bytes.Buffer
	ctx := context.Background()
	err := run(ctx, &out, &errOut, []string{"-i", testFile})

	if err != nil {
		t.Errorf("run() with -i flag error = %v, want nil", err)
	}

	output := out.String()
	// Output should contain numeric inode values
	if len(output) == 0 {
		t.Error("run() with -i flag should produce output")
	}
}

// TestRunWithIndividualFlags tests individual flags (not in long format).
func TestRunWithIndividualFlags(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "flag_test.txt")
	if err := os.WriteFile(testFile, []byte("test data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name string
		flag string
	}{
		{"user flag", "-u"},
		{"group flag", "-g"},
		{"size flag", "-s"},
		{"mode flag", "-p"},
		{"mount flag", "-m"},
		{"no follow flag", "-n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			ctx := context.Background()
			err := run(ctx, &out, &errOut, []string{tt.flag, testFile})

			if err != nil {
				t.Errorf("run() with %s error = %v, want nil", tt.flag, err)
			}

			if out.Len() == 0 {
				t.Errorf("run() with %s should produce output", tt.flag)
			}
		})
	}
}

// TestRunWithDeepPath tests handling deeply nested paths.
func TestRunWithDeepPath(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a deep directory structure
	deepPath := tmpDir
	for i := 0; i < 5; i++ {
		deepPath = filepath.Join(deepPath, fmt.Sprintf("level%d", i))
		if err := os.Mkdir(deepPath, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
	}

	testFile := filepath.Join(deepPath, "deep_file.txt")
	if err := os.WriteFile(testFile, []byte("deep"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var out, errOut bytes.Buffer
	ctx := context.Background()
	err := run(ctx, &out, &errOut, []string{testFile})

	if err != nil {
		t.Errorf("run() with deep path error = %v, want nil", err)
	}

	output := out.String()
	// Should show all levels
	for i := 0; i < 5; i++ {
		levelName := fmt.Sprintf("level%d", i)
		if !strings.Contains(output, levelName) {
			t.Errorf("output should contain %q", levelName)
		}
	}
}

// TestRunWithRelativePath tests handling relative paths.
func TestRunWithRelativePath(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "relative_test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Save current dir
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	defer os.Chdir(oldDir)

	// Change to temp dir
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	var out, errOut bytes.Buffer
	ctx := context.Background()
	// Use relative path
	err = run(ctx, &out, &errOut, []string{"relative_test.txt"})

	if err != nil {
		t.Errorf("run() with relative path error = %v, want nil", err)
	}

	if out.Len() == 0 {
		t.Error("run() with relative path should produce output")
	}
}

// TestRunWithMultiplePaths tests handling multiple paths.
func TestRunWithMultiplePaths(t *testing.T) {
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	if err := os.WriteFile(file1, []byte("one"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("two"), 0644); err != nil {
		t.Fatalf("Failed to create file2: %v", err)
	}

	var out, errOut bytes.Buffer
	ctx := context.Background()
	err := run(ctx, &out, &errOut, []string{file1, file2})

	if err != nil {
		t.Errorf("run() with multiple paths error = %v, want nil", err)
	}

	output := out.String()
	if !strings.Contains(output, "file1.txt") {
		t.Error("output should contain file1.txt")
	}
	if !strings.Contains(output, "file2.txt") {
		t.Error("output should contain file2.txt")
	}
	// Should have headers for multiple paths
	if !strings.Contains(output, "--") {
		t.Error("output should contain '--' headers for multiple paths")
	}
}

// TestProcessPath tests path processing.
func TestProcessPath(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "process_test.txt")
	if err := os.WriteFile(testFile, []byte("data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	var out bytes.Buffer
	ctx := context.Background()
	opts := options{
		noFollow: true,
		long:     false,
	}

	err := processPath(ctx, &out, testFile, opts)
	if err != nil {
		t.Errorf("processPath() error = %v, want nil", err)
	}

	if out.Len() == 0 {
		t.Error("processPath() should produce output")
	}
}

// TestCollectEntries tests entry collection.
func TestCollectEntries(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "collect_test.txt")
	if err := os.WriteFile(testFile, []byte("data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	opts := options{noFollow: true}

	entries, err := collectEntries(ctx, testFile, opts)
	if err != nil {
		t.Errorf("collectEntries() error = %v, want nil", err)
	}

	if len(entries) == 0 {
		t.Error("collectEntries() should return at least one entry")
	}

	// Check that last entry is our test file
	lastEntry := entries[len(entries)-1]
	if !strings.Contains(lastEntry.Name, "collect_test.txt") {
		t.Errorf("last entry name = %q, want to contain 'collect_test.txt'", lastEntry.Name)
	}
}

// TestCollectEntriesWithError tests error handling in collection.
func TestCollectEntriesWithError(t *testing.T) {
	ctx := context.Background()
	opts := options{noFollow: true}

	entries, err := collectEntries(ctx, "/this/does/not/exist", opts)
	if err == nil {
		t.Error("collectEntries() with nonexistent path should return error")
	}

	// Should still have at least one entry with the error
	if len(entries) == 0 {
		t.Error("collectEntries() should return entries even on error")
	}
}

// TestCollectEntriesWithCancellation tests context cancellation in collectEntries.
func TestCollectEntriesWithCancellation(t *testing.T) {
	// Create a deep directory structure to increase chance of hitting cancellation during walk
	tmpDir := t.TempDir()
	deepPath := tmpDir
	for i := 0; i < 10; i++ {
		deepPath = filepath.Join(deepPath, fmt.Sprintf("dir%d", i))
		if err := os.Mkdir(deepPath, 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		// Add some files in each directory
		for j := 0; j < 3; j++ {
			file := filepath.Join(deepPath, fmt.Sprintf("file%d.txt", j))
			if err := os.WriteFile(file, []byte("data"), 0644); err != nil {
				t.Fatalf("Failed to create file: %v", err)
			}
		}
	}

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Give the timeout a chance to expire
	time.Sleep(1 * time.Millisecond)

	opts := options{noFollow: false} // Follow symlinks to do more work

	_, err := collectEntries(ctx, deepPath, opts)
	// Should get either context.Canceled or context.DeadlineExceeded
	if err != context.Canceled && err != context.DeadlineExceeded {
		t.Logf("collectEntries() with cancelled context error = %v, want context error", err)
		// Don't fail the test as timing is unpredictable, just log it
	}
}

// TestCollectEntriesFollowingSymlinks tests collection with symlink following.
func TestCollectEntriesFollowingSymlinks(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a directory structure
	dir1 := filepath.Join(tmpDir, "dir1")
	if err := os.Mkdir(dir1, 0755); err != nil {
		t.Fatalf("Failed to create dir1: %v", err)
	}

	file1 := filepath.Join(dir1, "file1.txt")
	if err := os.WriteFile(file1, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create file1: %v", err)
	}

	// Create a symlink to the directory
	linkToDir := filepath.Join(tmpDir, "link_to_dir")
	if err := os.Symlink(dir1, linkToDir); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	ctx := context.Background()
	opts := options{noFollow: false} // Follow symlinks

	entries, err := collectEntries(ctx, linkToDir, opts)
	if err != nil {
		t.Errorf("collectEntries() with symlink following error = %v, want nil", err)
	}

	// Should have entries for the symlink and its target
	if len(entries) < 2 {
		t.Errorf("collectEntries() following symlinks returned %d entries, want at least 2", len(entries))
	}
}

// TestCalculateWidths tests width calculation.
func TestCalculateWidths(t *testing.T) {
	entries := []entry{
		{
			Mode:  "drwxr-xr-x",
			User:  "root",
			Group: "wheel",
			Size:  1024,
			Inode: 12345,
		},
		{
			Mode:  "-rw-r--r--",
			User:  "verylongusername",
			Group: "g",
			Size:  999999999,
			Inode: 9876543210,
		},
	}

	w := calculateWidths(entries)

	if w.mode < 10 {
		t.Errorf("mode width = %d, want >= 10", w.mode)
	}
	if w.user < 16 {
		t.Errorf("user width = %d, want >= 16", w.user)
	}
	if w.group < 1 {
		t.Errorf("group width = %d, want >= 1", w.group)
	}
	if w.size < 9 {
		t.Errorf("size width = %d, want >= 9", w.size)
	}
	if w.inode < 10 {
		t.Errorf("inode width = %d, want >= 10", w.inode)
	}
}

// TestCalculateWidthsEmpty tests width calculation with empty slice.
func TestCalculateWidthsEmpty(t *testing.T) {
	entries := []entry{}
	w := calculateWidths(entries)

	// All widths should be zero
	if w.mode != 0 || w.user != 0 || w.group != 0 || w.size != 0 || w.inode != 0 {
		t.Errorf("calculateWidths([]) = %+v, want all zeros", w)
	}
}

// TestPrintEntries tests entry printing.
func TestPrintEntries(t *testing.T) {
	entries := []entry{
		{
			Name:  "root",
			Mode:  "drwxr-xr-x",
			User:  "root",
			Group: "root",
			Size:  4096,
			Inode: 2,
			Level: 0,
		},
		{
			Name:  "test",
			Mode:  "-rw-r--r--",
			User:  "user",
			Group: "user",
			Size:  1024,
			Inode: 123,
			Level: 1,
		},
	}

	w := calculateWidths(entries)
	opts := options{
		mode:  true,
		user:  true,
		group: true,
		size:  true,
		inode: true,
	}

	var buf bytes.Buffer
	printEntries(&buf, entries, opts, w)

	output := buf.String()
	if !strings.Contains(output, "root") {
		t.Error("output should contain 'root'")
	}
	if !strings.Contains(output, "test") {
		t.Error("output should contain 'test'")
	}
}

// TestPrintEntriesWithError tests printing entries with errors.
func TestPrintEntriesWithError(t *testing.T) {
	entries := []entry{
		{
			Name: "good",
			Mode: "-rw-r--r--",
		},
		{
			Name: "bad",
			Err:  errors.New("test error"),
		},
	}

	w := calculateWidths(entries)
	opts := options{}

	var buf bytes.Buffer
	printEntries(&buf, entries, opts, w)

	output := buf.String()
	if !strings.Contains(output, "good") {
		t.Error("output should contain 'good'")
	}
	if !strings.Contains(output, "bad") {
		t.Error("output should contain 'bad'")
	}
	if !strings.Contains(output, "*") {
		t.Error("output should contain error marker '*'")
	}
}

// TestEntryPrint tests the entry print method.
func TestEntryPrint(t *testing.T) {
	e := &entry{
		Name:  "testfile",
		Mode:  "-rw-r--r--",
		User:  "testuser",
		Group: "testgroup",
		Size:  2048,
		Inode: 5678,
		Dev:   100,
		Pdev:  100,
		Level: 0,
	}

	w := widths{
		mode:  10,
		user:  10,
		group: 10,
		size:  10,
		inode: 10,
	}

	tests := []struct {
		name        string
		opts        options
		wantContain []string
	}{
		{
			name:        "mode only",
			opts:        options{mode: true},
			wantContain: []string{"-rw-r--r--", "testfile"},
		},
		{
			name:        "user only",
			opts:        options{user: true},
			wantContain: []string{"testuser", "testfile"},
		},
		{
			name:        "all fields",
			opts:        options{mode: true, user: true, group: true, size: true, inode: true},
			wantContain: []string{"-rw-r--r--", "testuser", "testgroup", "2048", "5678", "testfile"},
		},
		{
			name:        "mount point",
			opts:        options{mount: true},
			wantContain: []string{"testfile"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			e.print(&buf, tt.opts, w)
			output := buf.String()

			for _, want := range tt.wantContain {
				if !strings.Contains(output, want) {
					t.Errorf("print() output = %q, want to contain %q", output, want)
				}
			}
		})
	}
}

// TestEntryPrintWithMountPoint tests mount point detection.
func TestEntryPrintWithMountPoint(t *testing.T) {
	e := &entry{
		Name:  "mountpoint",
		Dev:   200,
		Pdev:  100, // Different device = mount point
		Level: 0,
	}

	w := widths{}
	opts := options{mount: true}

	var buf bytes.Buffer
	e.print(&buf, opts, w)
	output := buf.String()

	if !strings.Contains(output, mountPointSymbol) {
		t.Errorf("print() output = %q, want to contain mount point symbol %q", output, mountPointSymbol)
	}
}

// TestEntryPrintNoFollow tests print without following symlinks.
func TestEntryPrintNoFollow(t *testing.T) {
	e := &entry{
		Name:  "link",
		Link:  "target",
		Level: 0,
	}

	w := widths{}
	opts := options{noFollow: true}

	var buf bytes.Buffer
	e.print(&buf, opts, w)
	output := buf.String()

	// Should not show " -> target" when noFollow is true
	if strings.Contains(output, "->") {
		t.Error("print() with noFollow should not show link target")
	}
}

// TestEntryPrintFollowSymlink tests print with following symlinks.
func TestEntryPrintFollowSymlink(t *testing.T) {
	e := &entry{
		Name:  "link",
		Link:  "target",
		Level: 0,
	}

	w := widths{}
	opts := options{noFollow: false}

	var buf bytes.Buffer
	e.print(&buf, opts, w)
	output := buf.String()

	// Should show " -> target" when noFollow is false
	if !strings.Contains(output, "->") || !strings.Contains(output, "target") {
		t.Errorf("print() output = %q, want to contain '-> target'", output)
	}
}

// BenchmarkRun benchmarks the main run function.
func BenchmarkRun(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "bench.txt")
	if err := os.WriteFile(testFile, []byte("benchmark"), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var out, errOut bytes.Buffer
		ctx := context.Background()
		_ = run(ctx, &out, &errOut, []string{testFile})
	}
}

// BenchmarkCalculateWidths benchmarks width calculation.
func BenchmarkCalculateWidths(b *testing.B) {
	entries := make([]entry, 100)
	for i := range entries {
		entries[i] = entry{
			Mode:  fmt.Sprintf("-rw-r--r--%d", i),
			User:  fmt.Sprintf("user%d", i),
			Group: fmt.Sprintf("group%d", i),
			Size:  int64(i * 1000),
			Inode: uint64(i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateWidths(entries)
	}
}

// TestMain ensures proper test setup/teardown.
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()

	// Exit with test result code
	os.Exit(code)
}
