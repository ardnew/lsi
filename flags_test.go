package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantOpts  options
		wantPaths []string
		wantErr   bool
	}{
		{
			name: "no arguments",
			args: []string{},
			wantOpts: options{
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "version short flag",
			args: []string{"-v"},
			wantOpts: options{
				version: true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "version long flag",
			args: []string{"--version"},
			wantOpts: options{
				version: true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "timeout short flag",
			args: []string{"-t", "30s"},
			wantOpts: options{
				timeout: 30 * time.Second,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "timeout long flag",
			args: []string{"--timeout", "5m"},
			wantOpts: options{
				timeout: 5 * time.Minute,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "timeout with equals",
			args: []string{"--timeout=1h"},
			wantOpts: options{
				timeout: 1 * time.Hour,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "no-follow short flag",
			args: []string{"-n"},
			wantOpts: options{
				noFollow: true,
				timeout:  0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "no-follow long flag",
			args: []string{"--no-follow"},
			wantOpts: options{
				noFollow: true,
				timeout:  0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "long format short flag",
			args: []string{"-l"},
			wantOpts: options{
				long:    true,
				mode:    true,
				user:    true,
				group:   true,
				size:    true,
				mount:   true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "long format long flag",
			args: []string{"--long"},
			wantOpts: options{
				long:    true,
				mode:    true,
				user:    true,
				group:   true,
				size:    true,
				mount:   true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "permissions short flag",
			args: []string{"-p"},
			wantOpts: options{
				mode:    true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "permissions long flag",
			args: []string{"--permissions"},
			wantOpts: options{
				mode:    true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "user short flag",
			args: []string{"-u"},
			wantOpts: options{
				user:    true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "user long flag",
			args: []string{"--user"},
			wantOpts: options{
				user:    true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "group short flag",
			args: []string{"-g"},
			wantOpts: options{
				group:   true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "group long flag",
			args: []string{"--group"},
			wantOpts: options{
				group:   true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "size short flag",
			args: []string{"-s"},
			wantOpts: options{
				size:    true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "size long flag",
			args: []string{"--size"},
			wantOpts: options{
				size:    true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "inode short flag",
			args: []string{"-i"},
			wantOpts: options{
				inode:   true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "inode long flag",
			args: []string{"--inode"},
			wantOpts: options{
				inode:   true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "mount short flag",
			args: []string{"-m"},
			wantOpts: options{
				mount:   true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "mount long flag",
			args: []string{"--mount"},
			wantOpts: options{
				mount:   true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "single path",
			args: []string{"/tmp"},
			wantOpts: options{
				timeout: 0,
			},
			wantPaths: []string{"/tmp"},
			wantErr:   false,
		},
		{
			name: "multiple paths",
			args: []string{"/tmp", "/var", "/usr"},
			wantOpts: options{
				timeout: 0,
			},
			wantPaths: []string{"/tmp", "/var", "/usr"},
			wantErr:   false,
		},
		{
			name: "flags and paths",
			args: []string{"-l", "/tmp"},
			wantOpts: options{
				long:    true,
				mode:    true,
				user:    true,
				group:   true,
				size:    true,
				mount:   true,
				timeout: 0,
			},
			wantPaths: []string{"/tmp"},
			wantErr:   false,
		},
		{
			name: "double dash separator",
			args: []string{"-l", "--", "/tmp"},
			wantOpts: options{
				long:    true,
				mode:    true,
				user:    true,
				group:   true,
				size:    true,
				mount:   true,
				timeout: 0,
			},
			wantPaths: []string{"/tmp"},
			wantErr:   false,
		},
		{
			name: "multiple flags",
			args: []string{"-p", "-u", "-g", "-s", "-m"},
			wantOpts: options{
				mode:    true,
				user:    true,
				group:   true,
				size:    true,
				mount:   true,
				timeout: 0,
			},
			wantPaths: nil,
			wantErr:   false,
		},
		{
			name: "individual flags with path",
			args: []string{"-p", "-u", "-g", "-s", "-i", "/tmp"},
			wantOpts: options{
				mode:    true,
				user:    true,
				group:   true,
				size:    true,
				inode:   true,
				timeout: 0,
			},
			wantPaths: []string{"/tmp"},
			wantErr:   false,
		},
		{
			name: "timeout and path",
			args: []string{"-t", "30s", "/tmp"},
			wantOpts: options{
				timeout: 30 * time.Second,
			},
			wantPaths: []string{"/tmp"},
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOpts, gotPaths, err := parseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOpts != tt.wantOpts {
				t.Errorf("parseFlags() gotOpts = %+v, want %+v", gotOpts, tt.wantOpts)
			}
			if len(gotPaths) != len(tt.wantPaths) {
				t.Errorf("parseFlags() gotPaths length = %d, want %d", len(gotPaths), len(tt.wantPaths))
				return
			}
			for i := range gotPaths {
				if gotPaths[i] != tt.wantPaths[i] {
					t.Errorf("parseFlags() gotPaths[%d] = %v, want %v", i, gotPaths[i], tt.wantPaths[i])
				}
			}
		})
	}
}

func TestGetVersion(t *testing.T) {
	// getVersion() relies on build info which may not be available during tests.
	// Just verify it returns a non-empty string.
	version := getVersion()
	if version == "" {
		t.Error("getVersion() returned empty string")
	}
}

func TestAtob(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"true", true},
		{"True", true},
		{"TRUE", true},
		{"1", true},
		{"false", false},
		{"False", false},
		{"FALSE", false},
		{"0", false},
		{"", false},
		{"invalid", false},
		{"yes", false},
		{"no", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := atob(tt.input); got != tt.want {
				t.Errorf("atob(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestShowHelp(t *testing.T) {
	var buf bytes.Buffer
	showHelp(&buf)

	output := buf.String()

	// Check for essential help content
	expectedStrings := []string{
		"lsi - Analyze file paths",
		"Usage:",
		"Flags:",
		"-h --help",
		"-v --version",
		"-t --timeout",
		"-l --long",
		"-p --permissions",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("showHelp() output missing %q", expected)
		}
	}
}
