package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

// TestDetectShell tests shell detection from environment variables.
func TestDetectShell(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{
			name:     "bash from SHELL",
			env:      map[string]string{"SHELL": "/bin/bash"},
			expected: "bash",
		},
		{
			name:     "zsh from SHELL",
			env:      map[string]string{"SHELL": "/usr/bin/zsh"},
			expected: "zsh",
		},
		{
			name:     "fish from SHELL",
			env:      map[string]string{"SHELL": "/usr/local/bin/fish"},
			expected: "fish",
		},
		{
			name:     "zsh from ZSH_VERSION",
			env:      map[string]string{"ZSH_VERSION": "5.8"},
			expected: "zsh",
		},
		{
			name:     "bash from BASH_VERSION",
			env:      map[string]string{"BASH_VERSION": "5.0.0"},
			expected: "bash",
		},
		{
			name:     "fish from FISH_VERSION",
			env:      map[string]string{"FISH_VERSION": "3.3.0"},
			expected: "fish",
		},
		{
			name:     "powershell from PSModulePath",
			env:      map[string]string{"PSModulePath": "C:\\Program Files\\PowerShell\\Modules"},
			expected: "powershell",
		},
		{
			name:     "default to bash",
			env:      map[string]string{},
			expected: "bash",
		},
		{
			name:     "SHELL priority over version vars",
			env:      map[string]string{"SHELL": "/bin/bash", "ZSH_VERSION": "5.8"},
			expected: "bash",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			origEnv := make(map[string]string)
			envVars := []string{"SHELL", "ZSH_VERSION", "BASH_VERSION", "FISH_VERSION", "PSModulePath"}
			for _, key := range envVars {
				origEnv[key] = os.Getenv(key)
				os.Unsetenv(key)
			}
			defer func() {
				// Restore original environment
				for key, val := range origEnv {
					if val != "" {
						os.Setenv(key, val)
					} else {
						os.Unsetenv(key)
					}
				}
			}()

			// Set test environment
			for key, val := range tt.env {
				os.Setenv(key, val)
			}

			// Test detection
			got := detectShell()
			if got != tt.expected {
				t.Errorf("detectShell() = %q, want %q", got, tt.expected)
			}
		})
	}
}

// TestGenerateBashCompletion tests bash completion script generation.
func TestGenerateBashCompletion(t *testing.T) {
	var buf bytes.Buffer
	err := generateBashCompletion(&buf)
	if err != nil {
		t.Fatalf("generateBashCompletion() error = %v", err)
	}

	output := buf.String()

	// Check for essential bash completion elements
	required := []string{
		"_lsi_completion",
		"COMPREPLY",
		"complete -F _lsi_completion lsi",
		"--help",
		"--version",
		"--timeout",
		"--long",
	}

	for _, req := range required {
		if !strings.Contains(output, req) {
			t.Errorf("bash completion missing required string: %q", req)
		}
	}
}

// TestGenerateZshCompletion tests zsh completion script generation.
func TestGenerateZshCompletion(t *testing.T) {
	var buf bytes.Buffer
	err := generateZshCompletion(&buf)
	if err != nil {
		t.Fatalf("generateZshCompletion() error = %v", err)
	}

	output := buf.String()

	// Check for essential zsh completion elements
	required := []string{
		"#compdef lsi",
		"_lsi",
		"_arguments",
		"--help",
		"--version",
		"--timeout",
		"--long",
		"_files",
	}

	for _, req := range required {
		if !strings.Contains(output, req) {
			t.Errorf("zsh completion missing required string: %q", req)
		}
	}
}

// TestGenerateFishCompletion tests fish completion script generation.
func TestGenerateFishCompletion(t *testing.T) {
	var buf bytes.Buffer
	err := generateFishCompletion(&buf)
	if err != nil {
		t.Fatalf("generateFishCompletion() error = %v", err)
	}

	output := buf.String()

	// Check for essential fish completion elements
	required := []string{
		"complete -c lsi",
		"-s h -l help",
		"-s v -l version",
		"-s t -l timeout",
		"-s l -l long",
	}

	for _, req := range required {
		if !strings.Contains(output, req) {
			t.Errorf("fish completion missing required string: %q", req)
		}
	}
}

// TestGeneratePowershellCompletion tests PowerShell completion script generation.
func TestGeneratePowershellCompletion(t *testing.T) {
	var buf bytes.Buffer
	err := generatePowershellCompletion(&buf)
	if err != nil {
		t.Fatalf("generatePowershellCompletion() error = %v", err)
	}

	output := buf.String()

	// Check for essential PowerShell completion elements
	required := []string{
		"Register-ArgumentCompleter",
		"-CommandName lsi",
		"--help",
		"--version",
		"--timeout",
		"--long",
	}

	for _, req := range required {
		if !strings.Contains(output, req) {
			t.Errorf("powershell completion missing required string: %q", req)
		}
	}
}

// TestGenerateCompletion tests the main generateCompletion function.
func TestGenerateCompletion(t *testing.T) {
	tests := []struct {
		name      string
		shell     string
		wantErr   bool
		errString string
		contains  []string
	}{
		{
			name:     "bash",
			shell:    "bash",
			wantErr:  false,
			contains: []string{"_lsi_completion", "complete -F"},
		},
		{
			name:     "bash uppercase",
			shell:    "BASH",
			wantErr:  false,
			contains: []string{"_lsi_completion"},
		},
		{
			name:     "zsh",
			shell:    "zsh",
			wantErr:  false,
			contains: []string{"#compdef lsi", "_arguments"},
		},
		{
			name:     "fish",
			shell:    "fish",
			wantErr:  false,
			contains: []string{"complete -c lsi"},
		},
		{
			name:     "powershell",
			shell:    "powershell",
			wantErr:  false,
			contains: []string{"Register-ArgumentCompleter"},
		},
		{
			name:     "pwsh alias",
			shell:    "pwsh",
			wantErr:  false,
			contains: []string{"Register-ArgumentCompleter"},
		},
		{
			name:      "unsupported shell",
			shell:     "tcsh",
			wantErr:   true,
			errString: "unsupported shell: tcsh",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := generateCompletion(&buf, tt.shell)

			if tt.wantErr {
				if err == nil {
					t.Errorf("generateCompletion() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errString) {
					t.Errorf("generateCompletion() error = %q, want error containing %q", err.Error(), tt.errString)
				}
				return
			}

			if err != nil {
				t.Errorf("generateCompletion() unexpected error = %v", err)
				return
			}

			output := buf.String()
			for _, contains := range tt.contains {
				if !strings.Contains(output, contains) {
					t.Errorf("generateCompletion() output missing %q", contains)
				}
			}
		})
	}
}

// TestCompletionSubcommand tests the completion subcommand integration in run().
func TestCompletionSubcommand(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		contains []string
	}{
		{
			name:     "completion bash",
			args:     []string{"completion", "bash"},
			contains: []string{"_lsi_completion", "bash completion"},
		},
		{
			name:     "completion zsh",
			args:     []string{"completion", "zsh"},
			contains: []string{"#compdef lsi", "zsh completion"},
		},
		{
			name:     "completion fish",
			args:     []string{"completion", "fish"},
			contains: []string{"complete -c lsi", "fish completion"},
		},
		{
			name:     "completion powershell",
			args:     []string{"completion", "powershell"},
			contains: []string{"Register-ArgumentCompleter", "PowerShell completion"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			err := run(nil, &stdout, &stderr, tt.args)
			if err != nil {
				t.Errorf("run() error = %v", err)
				return
			}

			output := stdout.String()
			for _, contains := range tt.contains {
				if !strings.Contains(output, contains) {
					t.Errorf("run() output missing %q", contains)
				}
			}
		})
	}
}

// TestCompletionAutoDetect tests auto-detection when no shell specified.
func TestCompletionAutoDetect(t *testing.T) {
	// Save original SHELL environment variable
	origShell := os.Getenv("SHELL")
	defer func() {
		if origShell != "" {
			os.Setenv("SHELL", origShell)
		} else {
			os.Unsetenv("SHELL")
		}
	}()

	// Set SHELL to bash for predictable test
	os.Setenv("SHELL", "/bin/bash")

	var stdout, stderr bytes.Buffer
	err := run(nil, &stdout, &stderr, []string{"completion"})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "_lsi_completion") {
		t.Errorf("auto-detect should have generated bash completion, got: %s", output[:100])
	}
}

// TestCompletionUnsupportedShell tests error handling for unsupported shells.
func TestCompletionUnsupportedShell(t *testing.T) {
	var stdout, stderr bytes.Buffer
	err := run(nil, &stdout, &stderr, []string{"completion", "csh"})
	if err == nil {
		t.Errorf("run() expected error for unsupported shell, got nil")
		return
	}

	if !strings.Contains(err.Error(), "unsupported shell") {
		t.Errorf("run() error = %q, want error containing 'unsupported shell'", err.Error())
	}
}
