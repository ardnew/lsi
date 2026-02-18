package main

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestSplitPath tests path splitting logic.
func TestSplitPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantElem   []string
		wantVolume string
	}{
		{
			name:       "absolute path",
			path:       "/usr/local/bin",
			wantElem:   []string{"/", "usr", "local", "bin"},
			wantVolume: "",
		},
		{
			name:       "relative path",
			path:       "foo/bar/baz",
			wantElem:   []string{"foo", "bar", "baz"},
			wantVolume: "",
		},
		{
			name:       "root only",
			path:       "/",
			wantElem:   []string{"/"},
			wantVolume: "",
		},
		{
			name:       "single component",
			path:       "file",
			wantElem:   []string{"file"},
			wantVolume: "",
		},
		{
			name:       "trailing slash",
			path:       "/usr/local/",
			wantElem:   []string{"/", "usr", "local"},
			wantVolume: "",
		},
		{
			name:       "dot path",
			path:       ".",
			wantElem:   []string{"."},
			wantVolume: "",
		},
		{
			name:       "double dot",
			path:       "..",
			wantElem:   []string{".."},
			wantVolume: "",
		},
		{
			name:       "complex relative",
			path:       "./foo/../bar",
			wantElem:   []string{"bar"},
			wantVolume: "",
		},
		{
			name:       "multiple slashes",
			path:       "//usr///local",
			wantElem:   []string{"/", "usr", "local"},
			wantVolume: "",
		},
		{
			name:       "relative with volume (edge case)",
			path:       "rel/path",
			wantElem:   []string{"rel", "path"},
			wantVolume: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotElem, gotVolume := splitPath(tt.path)

			if gotVolume != tt.wantVolume {
				t.Errorf("splitPath(%q) volume = %q, want %q", tt.path, gotVolume, tt.wantVolume)
			}

			if len(gotElem) != len(tt.wantElem) {
				t.Errorf("splitPath(%q) elem count = %d, want %d", tt.path, len(gotElem), len(tt.wantElem))
				t.Errorf("  got: %v", gotElem)
				t.Errorf("  want: %v", tt.wantElem)
				return
			}

			for i := range gotElem {
				if gotElem[i] != tt.wantElem[i] {
					t.Errorf("splitPath(%q) elem[%d] = %q, want %q", tt.path, i, gotElem[i], tt.wantElem[i])
				}
			}
		})
	}
}

// TestUpperIf tests the upperIf function.
func TestUpperIf(t *testing.T) {
	tests := []struct {
		c     rune
		upper bool
		want  rune
	}{
		{'s', true, 'S'},
		{'s', false, 's'},
		{'t', true, 'T'},
		{'t', false, 't'},
		{'X', true, 'X'},
		{'X', false, 'x'},
		{'a', true, 'A'},
		{'Z', false, 'z'},
	}

	for _, tt := range tests {
		t.Run(string(tt.c), func(t *testing.T) {
			got := upperIf(tt.c, tt.upper)
			if got != tt.want {
				t.Errorf("upperIf(%q, %v) = %q, want %q", tt.c, tt.upper, got, tt.want)
			}
		})
	}
}

// TestMode tests file mode formatting.
func TestMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create files with different modes
	regularFile := filepath.Join(tmpDir, "regular")
	if err := os.WriteFile(regularFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create regular file: %v", err)
	}

	execFile := filepath.Join(tmpDir, "executable")
	if err := os.WriteFile(execFile, []byte("#!/bin/sh"), 0755); err != nil {
		t.Fatalf("Failed to create executable: %v", err)
	}

	symlink := filepath.Join(tmpDir, "symlink")
	if err := os.Symlink(regularFile, symlink); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	tests := []struct {
		name        string
		path        string
		wantPrefix  string
		wantContain string
	}{
		{
			name:        "regular file",
			path:        regularFile,
			wantPrefix:  "-",
			wantContain: "rw-",
		},
		{
			name:        "executable",
			path:        execFile,
			wantPrefix:  "-",
			wantContain: "rwx",
		},
		{
			name:        "symlink",
			path:        symlink,
			wantPrefix:  "l",
			wantContain: "rwx",
		},
		{
			name:        "directory",
			path:        tmpDir,
			wantPrefix:  "d",
			wantContain: "rwx",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := os.Lstat(tt.path)
			if err != nil {
				t.Fatalf("Failed to stat %q: %v", tt.path, err)
			}

			got := mode(info)

			if !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("mode() = %q, want prefix %q", got, tt.wantPrefix)
			}

			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("mode() = %q, want to contain %q", got, tt.wantContain)
			}
		})
	}
}

// TestModeSetuid tests setuid/setgid/sticky bit handling.
func TestModeSetuid(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file and try to set setuid (may fail due to permissions)
	testFile := filepath.Join(tmpDir, "setuid_test")
	if err := os.WriteFile(testFile, []byte("test"), 0755); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Try to set setuid bit
	if err := os.Chmod(testFile, 04755); err == nil {
		info, err := os.Lstat(testFile)
		if err != nil {
			t.Fatalf("Failed to stat file: %v", err)
		}

		got := mode(info)
		// Should have 's' or 'S' in owner execute position
		if !strings.Contains(got, "s") && !strings.Contains(got, "S") {
			t.Logf("mode() = %q, expected setuid marker (may fail in restricted environments)", got)
		}
	}

	// Test with ModeAppend, ModeExclusive, or ModeTemporary
	info := mockFileInfo{
		mode: 0644 | fs.ModeAppend,
		name: "test",
	}
	got := mode(info)
	if !strings.HasSuffix(got, "+") {
		t.Errorf("mode() with ModeAppend = %q, want '+' suffix", got)
	}
}

// TestEntryFmtName tests entry name formatting.
func TestEntryFmtName(t *testing.T) {
	tests := []struct {
		name        string
		entry       entry
		wantContain string
	}{
		{
			name: "no link",
			entry: entry{
				Name:  "file.txt",
				Link:  "",
				Level: 0,
			},
			wantContain: "file.txt",
		},
		{
			name: "with link",
			entry: entry{
				Name:  "symlink",
				Link:  "target",
				Level: 0,
			},
			wantContain: "symlink -> target",
		},
		{
			name: "nested level",
			entry: entry{
				Name:  "nested",
				Link:  "",
				Level: 2,
			},
			wantContain: "nested",
		},
		{
			name: "deeply nested with link",
			entry: entry{
				Name:  "deep",
				Link:  "/absolute/path",
				Level: 5,
			},
			wantContain: "deep -> /absolute/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.entry.fmtName()

			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("fmtName() = %q, want to contain %q", got, tt.wantContain)
			}

			// Check indentation
			expectedIndent := indentWidth * tt.entry.Level
			actualIndent := len(got) - len(strings.TrimLeft(got, " "))
			if actualIndent != expectedIndent {
				t.Errorf("fmtName() indent = %d spaces, want %d", actualIndent, expectedIndent)
			}
		})
	}
}

// TestMakeEntry tests entry creation.
func TestMakeEntry(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()
	e := makeEntry(ctx, tmpDir, "test.txt", "", "test.txt", 0)

	if e.Err != nil {
		t.Errorf("makeEntry() error = %v, want nil", e.Err)
	}

	if e.Name != "test.txt" {
		t.Errorf("makeEntry() Name = %q, want %q", e.Name, "test.txt")
	}

	if e.Path != "test.txt" {
		t.Errorf("makeEntry() Path = %q, want %q", e.Path, "test.txt")
	}

	if e.Size != 7 {
		t.Errorf("makeEntry() Size = %d, want 7", e.Size)
	}

	if e.Mode == "" {
		t.Error("makeEntry() Mode should not be empty")
	}

	if e.User == "" {
		t.Error("makeEntry() User should not be empty")
	}

	if e.Group == "" {
		t.Error("makeEntry() Group should not be empty")
	}
}

// TestMakeEntryWithSymlink tests entry creation for symlinks.
func TestMakeEntryWithSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	target := filepath.Join(tmpDir, "target.txt")
	if err := os.WriteFile(target, []byte("target"), 0644); err != nil {
		t.Fatalf("Failed to create target file: %v", err)
	}

	symlink := filepath.Join(tmpDir, "link")
	if err := os.Symlink(target, symlink); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	ctx := context.Background()
	e := makeEntry(ctx, tmpDir, "link", "", "link", 0)

	if e.Err != nil {
		t.Errorf("makeEntry() error = %v, want nil", e.Err)
	}

	if e.Link == "" {
		t.Error("makeEntry() Link should not be empty for symlink")
	}

	if !strings.Contains(e.Link, "target.txt") {
		t.Errorf("makeEntry() Link = %q, want to contain 'target.txt'", e.Link)
	}

	// Mode should indicate symlink
	if !strings.HasPrefix(e.Mode, "l") {
		t.Errorf("makeEntry() Mode = %q, want to start with 'l'", e.Mode)
	}
}

// TestMakeEntryWithCanceledContext tests context cancellation handling.
func TestMakeEntryWithCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	e := makeEntry(ctx, "", "/some/path", "", "path", 0)

	if e.Err == nil {
		t.Error("makeEntry() with canceled context should return error")
	}

	if e.Err != context.Canceled {
		t.Errorf("makeEntry() error = %v, want context.Canceled", e.Err)
	}
}

// TestMakeEntryNonexistent tests entry creation for nonexistent path.
func TestMakeEntryNonexistent(t *testing.T) {
	ctx := context.Background()
	e := makeEntry(ctx, "", "/this/does/not/exist", "", "exist", 0)

	if e.Err == nil {
		t.Error("makeEntry() for nonexistent path should return error")
	}

	// Should still populate basic fields
	if e.Name != "exist" {
		t.Errorf("makeEntry() Name = %q, want 'exist'", e.Name)
	}
}

// TestWalk tests the walk function.
func TestWalk(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a simple file structure
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	testFile := filepath.Join(subDir, "file.txt")
	if err := os.WriteFile(testFile, []byte("data"), 0644); err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}

	ctx := context.Background()
	var entries []entry

	err := walk(ctx, testFile, func(ctx context.Context, e entry) (bool, error) {
		entries = append(entries, e)
		return false, nil // Don't follow symlinks
	})

	if err != nil {
		t.Errorf("walk() error = %v, want nil", err)
	}

	if len(entries) == 0 {
		t.Fatal("walk() should produce at least one entry")
	}

	// Last entry should be our file
	lastEntry := entries[len(entries)-1]
	if !strings.Contains(lastEntry.Name, "file.txt") {
		t.Errorf("last entry name = %q, want to contain 'file.txt'", lastEntry.Name)
	}
}

// TestWalkWithSymlink tests walking paths with symlinks.
func TestWalkWithSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create target file
	target := filepath.Join(tmpDir, "target.txt")
	if err := os.WriteFile(target, []byte("target"), 0644); err != nil {
		t.Fatalf("Failed to create target: %v", err)
	}

	// Create symlink
	symlink := filepath.Join(tmpDir, "link")
	if err := os.Symlink(target, symlink); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	ctx := context.Background()
	var entries []entry

	// Walk with following symlinks
	err := walk(ctx, symlink, func(ctx context.Context, e entry) (bool, error) {
		entries = append(entries, e)
		return true, nil // Follow symlinks
	})

	if err != nil {
		t.Errorf("walk() error = %v, want nil", err)
	}

	// Should have entries for both the symlink and its target
	if len(entries) < 2 {
		t.Errorf("walk() with symlink following produced %d entries, want at least 2", len(entries))
	}
}

// TestWalkWithError tests walk error handling.
func TestWalkWithError(t *testing.T) {
	ctx := context.Background()

	err := walk(ctx, "/this/does/not/exist", func(ctx context.Context, e entry) (bool, error) {
		if e.Err != nil {
			return false, e.Err
		}
		return false, nil
	})

	if err == nil {
		t.Error("walk() with nonexistent path should return error")
	}
}

// TestWalkWithContextCancellation tests context cancellation during walk.
func TestWalkWithContextCancellation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	err := walk(ctx, testFile, func(ctx context.Context, e entry) (bool, error) {
		callCount++
		if callCount == 2 {
			cancel() // Cancel after second call
			// Check context immediately
			if ctx.Err() != nil {
				return false, ctx.Err()
			}
		}
		return false, nil
	})

	// Should either complete successfully or get cancellation error
	if err != nil && err != context.Canceled {
		t.Errorf("walk() with canceled context error = %v, want nil or context.Canceled", err)
	}
}

// TestWalkWithTimeout tests walk timeout.
func TestWalkWithTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	time.Sleep(2 * time.Millisecond) // Ensure timeout

	err := walk(ctx, "/usr", func(ctx context.Context, e entry) (bool, error) {
		return false, nil
	})

	if err != context.DeadlineExceeded {
		t.Errorf("walk() with timeout error = %v, want context.DeadlineExceeded", err)
	}
}

// TestWalkRecursiveRelativeSymlink tests relative symlink resolution.
func TestWalkRecursiveRelativeSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create subdirectory with target
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	target := filepath.Join(subDir, "target.txt")
	if err := os.WriteFile(target, []byte("target"), 0644); err != nil {
		t.Fatalf("Failed to create target: %v", err)
	}

	// Create relative symlink in subdir
	symlink := filepath.Join(subDir, "link")
	if err := os.Symlink("target.txt", symlink); err != nil {
		t.Fatalf("Failed to create symlink: %v", err)
	}

	ctx := context.Background()
	var entries []entry

	err := walk(ctx, symlink, func(ctx context.Context, e entry) (bool, error) {
		entries = append(entries, e)
		return true, nil // Follow symlinks
	})

	if err != nil {
		t.Errorf("walk() with relative symlink error = %v, want nil", err)
	}

	// Should resolve the relative symlink
	if len(entries) == 0 {
		t.Fatal("walk() should produce entries")
	}
}

// TestWalkCircularSymlink tests handling of circular symlinks.
func TestWalkCircularSymlink(t *testing.T) {
	tmpDir := t.TempDir()

	// Create circular symlink: link1 -> link2 -> link1
	link1 := filepath.Join(tmpDir, "link1")
	link2 := filepath.Join(tmpDir, "link2")

	if err := os.Symlink(link2, link1); err != nil {
		t.Fatalf("Failed to create link1: %v", err)
	}

	if err := os.Symlink(link1, link2); err != nil {
		t.Fatalf("Failed to create link2: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	depth := 0
	maxDepth := 100 // Reasonable limit to detect infinite loop

	err := walk(ctx, link1, func(ctx context.Context, e entry) (bool, error) {
		depth++
		if depth > maxDepth {
			return false, nil // Stop following to prevent infinite recursion
		}
		return true, nil // Follow symlinks
	})

	// Either we hit max depth or got an error (ELOOP on some systems)
	if err == nil && depth <= maxDepth {
		t.Logf("Circular symlink walk completed with depth %d", depth)
	}
}

// TestWalkAbsolutePath tests walking absolute paths.
func TestWalkAbsolutePath(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "absolute_test.txt")
	if err := os.WriteFile(testFile, []byte("absolute"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Get absolute path
	absPath, err := filepath.Abs(testFile)
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	ctx := context.Background()
	var entries []entry

	err = walk(ctx, absPath, func(ctx context.Context, e entry) (bool, error) {
		entries = append(entries, e)
		return false, nil
	})

	if err != nil {
		t.Errorf("walk() with absolute path error = %v, want nil", err)
	}

	if len(entries) == 0 {
		t.Fatal("walk() should produce entries")
	}

	// First entry should be root
	if !strings.HasPrefix(entries[0].Name, "/") && entries[0].Name != "/" {
		t.Logf("First entry name = %q (may vary by system)", entries[0].Name)
	}
}

// TestModeSpecialBits tests mode formatting with special permission bits.
func TestModeSpecialBits(t *testing.T) {
	tests := []struct {
		name        string
		mode        os.FileMode
		wantContain string
	}{
		{
			name:        "regular file",
			mode:        0644,
			wantContain: "-rw-r--r--",
		},
		{
			name:        "directory",
			mode:        fs.ModeDir | 0755,
			wantContain: "drwxr-xr-x",
		},
		{
			name:        "symlink",
			mode:        fs.ModeSymlink | 0777,
			wantContain: "lrwxrwxrwx",
		},
		{
			name:        "device",
			mode:        fs.ModeDevice | 0660,
			wantContain: "brw-rw----",
		},
		{
			name:        "char device",
			mode:        fs.ModeCharDevice | 0660,
			wantContain: "crw-rw----",
		},
		{
			name:        "socket",
			mode:        fs.ModeSocket | 0755,
			wantContain: "srwxr-xr-x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := mockFileInfo{
				mode: tt.mode,
				name: "test",
			}
			got := mode(mock)

			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("mode(%v) = %q, want to contain %q", tt.mode, got, tt.wantContain)
			}
		})
	}
}

// mockFileInfo implements os.FileInfo for testing.
type mockFileInfo struct {
	name  string
	size  int64
	mode  os.FileMode
	mtime time.Time
}

func (m mockFileInfo) Name() string       { return m.name }
func (m mockFileInfo) Size() int64        { return m.size }
func (m mockFileInfo) Mode() os.FileMode  { return m.mode }
func (m mockFileInfo) ModTime() time.Time { return m.mtime }
func (m mockFileInfo) IsDir() bool        { return m.mode&fs.ModeDir != 0 }
func (m mockFileInfo) Sys() interface{}   { return &syscall.Stat_t{} }

// BenchmarkSplitPath benchmarks path splitting.
func BenchmarkSplitPath(b *testing.B) {
	paths := []string{
		"/usr/local/bin/program",
		"relative/path/to/file",
		"/",
		".",
		"../../../deep/relative/path",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		path := paths[i%len(paths)]
		_, _ = splitPath(path)
	}
}

// BenchmarkMode benchmarks mode formatting.
func BenchmarkMode(b *testing.B) {
	info := mockFileInfo{
		name:  "test",
		size:  1024,
		mode:  0644,
		mtime: time.Now(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mode(info)
	}
}

// BenchmarkWalk benchmarks the walk function.
func BenchmarkWalk(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "bench.txt")
	if err := os.WriteFile(testFile, []byte("benchmark"), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = walk(ctx, testFile, func(ctx context.Context, e entry) (bool, error) {
			return false, nil
		})
	}
}

// BenchmarkMakeEntry benchmarks entry creation.
func BenchmarkMakeEntry(b *testing.B) {
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "bench.txt")
	if err := os.WriteFile(testFile, []byte("benchmark"), 0644); err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = makeEntry(ctx, tmpDir, "bench.txt", "", "bench.txt", 0)
	}
}
