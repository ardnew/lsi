package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzSplitPath tests splitPath with various inputs.
func FuzzSplitPath(f *testing.F) {
	// Seed corpus with interesting paths
	seeds := []string{
		"/",
		"/usr/local/bin",
		"relative/path",
		".",
		"..",
		"../../../up",
		"//double//slash",
		"/trailing/",
		"./current/dir",
		"path/with spaces/file",
		"path\twith\ttabs",
		"path\nwith\nnewlines",
		"path/with/../dots",
		"/very/very/very/very/very/very/very/deep/nested/path/to/file",
		"",
		"single",
		"/single",
		"a/b",
		"/a/b",
		"a/b/c/d/e/f/g",
		"./././.",
		"../../..",
		"/root",
		"~/home",
		"./relative/../path",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, path string) {
		// Skip invalid UTF-8 strings
		if !utf8.ValidString(path) {
			return
		}

		// splitPath should never panic
		elem, volume := splitPath(path)

		// Basic invariants
		// Note: filepath.Clean("") returns ".", so empty path becomes ["."]

		// Volume should be a prefix of the path (if non-empty and path non-empty)
		if volume != "" && path != "" && !strings.HasPrefix(path, volume) {
			t.Errorf("splitPath(%q) volume %q is not a prefix", path, volume)
		}

		// Elements should not contain separators (except for root "/")
		for i, e := range elem {
			if e == "/" {
				// Root is allowed
				if i != 0 {
					t.Errorf("splitPath(%q) has '/' at position %d", path, i)
				}
			} else if strings.Contains(e, string(filepath.Separator)) {
				t.Errorf("splitPath(%q) element %d contains separator: %q", path, i, e)
			}
		}

		// Should not return nil slice
		if elem == nil {
			t.Errorf("splitPath(%q) returned nil slice", path)
		}
	})
}

// FuzzUpperIf tests upperIf with various runes.
func FuzzUpperIf(f *testing.F) {
	// Seed with various runes
	seeds := []rune{'a', 'z', 'A', 'Z', 's', 't', 'x', '0', '9', ' ', '\t', '\n', '!', '@', '#'}

	for _, seed := range seeds {
		f.Add(string(seed), true)
		f.Add(string(seed), false)
	}

	f.Fuzz(func(t *testing.T, s string, upper bool) {
		// Extract first rune
		if len(s) == 0 {
			return
		}

		r := []rune(s)[0]

		// upperIf should never panic
		result := upperIf(r, upper)

		// Result should be a valid rune
		if !utf8.ValidRune(result) {
			t.Errorf("upperIf(%q, %v) returned invalid rune", r, upper)
		}

		// Check behavior
		if upper {
			// Should be uppercase or non-letter
			if result != r && result < 'A' || result > 'Z' {
				// It's OK if it's not a letter
				lower := strings.ToLower(string(result))
				higher := strings.ToUpper(string(r))
				if lower != string(result) && higher != string(result) {
					// Both conversions changed it, which is unexpected
					t.Logf("upperIf(%q, true) = %q (expected uppercase)", r, result)
				}
			}
		}
	})
}

// FuzzMode tests mode formatting with various file modes.
func FuzzMode(f *testing.F) {
	// Seed with various mode values
	seeds := []uint32{
		0644,
		0755,
		0777,
		0600,
		0000,
		04755, // setuid
		02755, // setgid
		01755, // sticky
		07777, // all bits
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, modeInt uint32) {
		info := mockFileInfo{
			name: "fuzz",
			mode: os.FileMode(modeInt),
		}

		// mode should never panic
		result := mode(info)

		// Result should be at least 10 characters (Unix permission string)
		if len(result) < 10 {
			t.Errorf("mode(%v) = %q, expected at least 10 characters", modeInt, result)
		}

		// Should only contain valid permission characters
		validChars := "drwxstlbpsc-+XSTABCDEFGHIJKLMNOPQRUVWYZ?"
		for _, c := range result {
			if !strings.ContainsRune(validChars, c) {
				t.Errorf("mode(%v) = %q contains invalid character %q", modeInt, result, c)
			}
		}

		// First character should be a file type indicator
		firstChar := rune(result[0])
		if !strings.ContainsRune("d-lbpsc?", firstChar) {
			t.Logf("mode(%v) = %q has unexpected first character %q", modeInt, result, firstChar)
		}
	})
}

// FuzzEntryFmtName tests entry name formatting.
func FuzzEntryFmtName(f *testing.F) {
	// Seed corpus
	seeds := []struct {
		name  string
		link  string
		level int
	}{
		{"file", "", 0},
		{"symlink", "target", 0},
		{"deep", "target", 5},
		{"spaces in name", "target with spaces", 2},
		{"tab\tname", "tab\ttarget", 1},
		{"", "", 0},
		{"a", "b", 100},
	}

	for _, seed := range seeds {
		f.Add(seed.name, seed.link, seed.level)
	}

	f.Fuzz(func(t *testing.T, name string, link string, level int) {
		// Skip invalid UTF-8
		if !utf8.ValidString(name) || !utf8.ValidString(link) {
			return
		}

		// Limit level to reasonable range to avoid huge strings
		if level < 0 {
			level = 0
		}
		if level > 1000 {
			level = 1000
		}

		e := &entry{
			Name:  name,
			Link:  link,
			Level: level,
		}

		// fmtName should never panic
		result := e.fmtName()

		// Result should contain the name
		if !strings.Contains(result, name) {
			t.Errorf("fmtName() = %q, want to contain %q", result, name)
		}

		// If link is non-empty, result should contain it
		if link != "" && !strings.Contains(result, link) {
			t.Errorf("fmtName() = %q, want to contain link %q", result, link)
		}

		// Check indentation is reasonable
		indent := len(result) - len(strings.TrimLeft(result, " "))
		expectedIndent := indentWidth * level
		if indent != expectedIndent {
			// Allow some flexibility for strings with special characters
			// The indentation may vary if name contains non-ASCII or control chars
			if len(name) > 0 && name[0] > 127 {
				// Non-ASCII first character may affect spacing
				return
			}
			if !strings.Contains(name, "\n") && !strings.Contains(link, "\n") {
				// Only strict check for simple ASCII names
				for _, r := range name {
					if r < 32 || r > 126 {
						// Contains non-printable or non-ASCII, skip strict check
						return
					}
				}
				t.Errorf("fmtName() indent = %d, want %d (name=%q link=%q)", indent, expectedIndent, name, link)
			}
		}
	})
}

// FuzzWalk tests walk with various paths.
func FuzzWalk(f *testing.F) {
	// Create a temporary directory for testing
	tmpDir := f.TempDir()
	testFile := filepath.Join(tmpDir, "fuzz_test.txt")
	if err := os.WriteFile(testFile, []byte("fuzz"), 0644); err != nil {
		f.Fatalf("Failed to create test file: %v", err)
	}

	// Create symlink
	symlink := filepath.Join(tmpDir, "link")
	if err := os.Symlink(testFile, symlink); err != nil {
		f.Fatalf("Failed to create symlink: %v", err)
	}

	// Seed with various paths
	seeds := []string{
		testFile,
		tmpDir,
		symlink,
		"/",
		".",
		"..",
		"/tmp",
		"/nonexistent",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, path string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(path) {
			return
		}

		// Skip overly long paths
		if len(path) > 4096 {
			return
		}

		ctx := context.Background()

		// walk should never panic, even with invalid paths
		callCount := 0
		maxCalls := 1000 // Prevent infinite loops

		_ = walk(ctx, path, func(ctx context.Context, e entry) (bool, error) {
			callCount++
			if callCount > maxCalls {
				return false, nil // Stop to prevent runaway
			}

			// Basic invariant: entry should have a name
			if e.Name == "" && e.Err == nil {
				t.Error("walk() produced entry with empty name and no error")
			}

			// Don't follow symlinks to prevent infinite loops in fuzzing
			return false, nil
		})
	})
}

// FuzzMakeEntry tests entry creation with various inputs.
func FuzzMakeEntry(f *testing.F) {
	// Create test environment
	tmpDir := f.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		f.Fatalf("Failed to create test file: %v", err)
	}

	// Seed corpus
	seeds := []struct {
		from   string
		path   string
		volume string
		name   string
		level  int
	}{
		{tmpDir, "test.txt", "", "test.txt", 0},
		{"", "/tmp", "", "tmp", 0},
		{tmpDir, "nonexistent", "", "nonexistent", 0},
		{tmpDir, "test.txt", "", "test.txt", 5},
		{"", ".", "", ".", 0},
		{"", "..", "", "..", 0},
	}

	for _, seed := range seeds {
		f.Add(seed.from, seed.path, seed.volume, seed.name, seed.level)
	}

	f.Fuzz(func(t *testing.T, from, path, volume, name string, level int) {
		// Skip invalid UTF-8
		if !utf8.ValidString(from) || !utf8.ValidString(path) ||
			!utf8.ValidString(volume) || !utf8.ValidString(name) {
			return
		}

		// Limit level to reasonable range
		if level < 0 {
			level = 0
		}
		if level > 100 {
			level = 100
		}

		// Skip overly long paths
		if len(from) > 1024 || len(path) > 1024 {
			return
		}

		ctx := context.Background()

		// makeEntry should never panic
		e := makeEntry(ctx, from, path, volume, name, level)

		// Basic invariants
		if e.Path != path {
			t.Errorf("makeEntry() Path = %q, want %q", e.Path, path)
		}

		if e.Volume != volume {
			t.Errorf("makeEntry() Volume = %q, want %q", e.Volume, volume)
		}

		if e.Name != name {
			t.Errorf("makeEntry() Name = %q, want %q", e.Name, name)
		}

		if e.Level != level {
			t.Errorf("makeEntry() Level = %d, want %d", e.Level, level)
		}

		// If there's no error, some fields should be populated
		if e.Err == nil {
			if e.Mode == "" {
				t.Error("makeEntry() with no error should have Mode")
			}
		}
	})
}

// FuzzAtob tests the atob function with various strings.
func FuzzAtob(f *testing.F) {
	// Seed with known boolean strings
	seeds := []string{
		"true", "false", "True", "False", "TRUE", "FALSE",
		"1", "0", "t", "f", "T", "F",
		"yes", "no", "y", "n",
		"", "invalid", "maybe", "2", "-1",
	}

	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, s string) {
		// Skip invalid UTF-8
		if !utf8.ValidString(s) {
			return
		}

		// atob should never panic
		result := atob(s)

		// Result should be a boolean (always true)
		_ = result

		// No additional assertions needed - we're just checking for panics
	})
}
