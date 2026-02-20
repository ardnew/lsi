package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// detectShell determines the user's current shell from environment variables.
// It checks $SHELL first, then falls back to other shell indicators.
// Returns one of: "bash", "zsh", "fish", "powershell", or "bash" as default.
func detectShell() string {
	// Check $SHELL environment variable (Unix/Linux/macOS)
	if shell := os.Getenv("SHELL"); shell != "" {
		base := filepath.Base(shell)
		switch base {
		case "bash":
			return "bash"
		case "zsh":
			return "zsh"
		case "fish":
			return "fish"
		}
	}

	// Check for PowerShell on Windows
	if psModulePath := os.Getenv("PSModulePath"); psModulePath != "" {
		return "powershell"
	}

	// Check ZSH_VERSION (set by zsh)
	if os.Getenv("ZSH_VERSION") != "" {
		return "zsh"
	}

	// Check BASH_VERSION (set by bash)
	if os.Getenv("BASH_VERSION") != "" {
		return "bash"
	}

	// Check FISH_VERSION (set by fish)
	if os.Getenv("FISH_VERSION") != "" {
		return "fish"
	}

	// Default to bash (most common)
	return "bash"
}

// generateCompletion generates shell completion script for the specified shell.
// Supported shells: bash, zsh, fish, powershell.
func generateCompletion(w io.Writer, shell string) error {
	shell = strings.ToLower(shell)

	switch shell {
	case "bash":
		return generateBashCompletion(w)
	case "zsh":
		return generateZshCompletion(w)
	case "fish":
		return generateFishCompletion(w)
	case "powershell", "pwsh":
		return generatePowershellCompletion(w)
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish, powershell)", shell)
	}
}

// generateBashCompletion generates bash completion script.
func generateBashCompletion(w io.Writer) error {
	script := `# bash completion for lsi
# Source this file to enable completion:
#   source <(lsi completion bash)
# Or add to ~/.bashrc:
#   eval "$(lsi completion bash)"

_lsi_completion() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    
    # All available flags
    opts="-h --help -v --version -t --timeout -n --no-follow -l --long -p --permissions -u --user -g --group -s --size -i --inode -m --mount"
    
    # Handle timeout flag requiring a value
    if [[ "${prev}" == "-t" || "${prev}" == "--timeout" ]]; then
        # Suggest common timeout values
        COMPREPLY=( $(compgen -W "30s 1m 5m 10m" -- "${cur}") )
        return 0
    fi
    
    # Complete flags
    if [[ "${cur}" == -* ]]; then
        COMPREPLY=( $(compgen -W "${opts}" -- "${cur}") )
        return 0
    fi
    
    # Complete file paths
    COMPREPLY=( $(compgen -f -- "${cur}") )
    return 0
}

# Register the completion function
complete -F _lsi_completion lsi
`
	_, err := fmt.Fprint(w, script)
	return err
}

// generateZshCompletion generates zsh completion script.
func generateZshCompletion(w io.Writer) error {
	script := `#compdef lsi
# zsh completion for lsi
# Add to your ~/.zshrc:
#   eval "$(lsi completion zsh)"
# Or save to a file in your $fpath:
#   lsi completion zsh > ~/.zsh/completions/_lsi

_lsi() {
    local -a opts
    opts=(
        '(-h --help)'{-h,--help}'[Display help message]'
        '(-v --version)'{-v,--version}'[Display version information]'
        '(-t --timeout)'{-t,--timeout}'[Timeout duration (e.g., 30s, 5m)]:duration:(30s 1m 5m 10m)'
        '(-n --no-follow)'{-n,--no-follow}'[Do not follow symlinks]'
        '(-l --long)'{-l,--long}'[Output using long format]'
        '(-p --permissions)'{-p,--permissions}'[Output file type and permissions]'
        '(-u --user)'{-u,--user}'[Output file owner]'
        '(-g --group)'{-g,--group}'[Output file group]'
        '(-s --size)'{-s,--size}'[Output file size (bytes)]'
        '(-i --inode)'{-i,--inode}'[Output file inode]'
        '(-m --mount)'{-m,--mount}'[Output mount point symbols]'
        '*:file:_files'
    )
    
    _arguments -s -S $opts
}

_lsi "$@"
`
	_, err := fmt.Fprint(w, script)
	return err
}

// generateFishCompletion generates fish completion script.
func generateFishCompletion(w io.Writer) error {
	script := `# fish completion for lsi
# Save to ~/.config/fish/completions/lsi.fish
# Or run: lsi completion fish > ~/.config/fish/completions/lsi.fish

# Remove any existing completions
complete -c lsi -e

# Flag completions
complete -c lsi -s h -l help -d 'Display help message'
complete -c lsi -s v -l version -d 'Display version information'
complete -c lsi -s t -l timeout -d 'Timeout duration' -x -a '30s 1m 5m 10m'
complete -c lsi -s n -l no-follow -d 'Do not follow symlinks'
complete -c lsi -s l -l long -d 'Output using long format'
complete -c lsi -s p -l permissions -d 'Output file type and permissions'
complete -c lsi -s u -l user -d 'Output file owner'
complete -c lsi -s g -l group -d 'Output file group'
complete -c lsi -s s -l size -d 'Output file size (bytes)'
complete -c lsi -s i -l inode -d 'Output file inode'
complete -c lsi -s m -l mount -d 'Output mount point symbols'

# File path completion (default behavior)
complete -c lsi -f -a '(__fish_complete_path)'
`
	_, err := fmt.Fprint(w, script)
	return err
}

// generatePowershellCompletion generates PowerShell completion script.
func generatePowershellCompletion(w io.Writer) error {
	script := `# PowerShell completion for lsi
# Add to your PowerShell profile:
#   lsi completion powershell | Out-String | Invoke-Expression
# Or add this line to your profile:
#   Invoke-Expression -Command $(lsi completion powershell | Out-String)

Register-ArgumentCompleter -CommandName lsi -ScriptBlock {
    param($commandName, $wordToComplete, $commandAst, $fakeBoundParameters)
    
    $flags = @(
        @{ Name = '-h'; Description = 'Display help message' }
        @{ Name = '--help'; Description = 'Display help message' }
        @{ Name = '-v'; Description = 'Display version information' }
        @{ Name = '--version'; Description = 'Display version information' }
        @{ Name = '-t'; Description = 'Timeout duration (e.g., 30s, 5m)' }
        @{ Name = '--timeout'; Description = 'Timeout duration (e.g., 30s, 5m)' }
        @{ Name = '-n'; Description = 'Do not follow symlinks' }
        @{ Name = '--no-follow'; Description = 'Do not follow symlinks' }
        @{ Name = '-l'; Description = 'Output using long format' }
        @{ Name = '--long'; Description = 'Output using long format' }
        @{ Name = '-p'; Description = 'Output file type and permissions' }
        @{ Name = '--permissions'; Description = 'Output file type and permissions' }
        @{ Name = '-u'; Description = 'Output file owner' }
        @{ Name = '--user'; Description = 'Output file owner' }
        @{ Name = '-g'; Description = 'Output file group' }
        @{ Name = '--group'; Description = 'Output file group' }
        @{ Name = '-s'; Description = 'Output file size (bytes)' }
        @{ Name = '--size'; Description = 'Output file size (bytes)' }
        @{ Name = '-i'; Description = 'Output file inode' }
        @{ Name = '--inode'; Description = 'Output file inode' }
        @{ Name = '-m'; Description = 'Output mount point symbols' }
        @{ Name = '--mount'; Description = 'Output mount point symbols' }
    )
    
    # Check if completing a timeout value
    $prevWord = $commandAst.CommandElements[-2].ToString()
    if ($prevWord -eq '-t' -or $prevWord -eq '--timeout') {
        $timeouts = @('30s', '1m', '5m', '10m')
        $timeouts | Where-Object { $_ -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_, $_, 'ParameterValue', $_)
        }
        return
    }
    
    # Complete flags
    if ($wordToComplete -match '^-') {
        $flags | Where-Object { $_.Name -like "$wordToComplete*" } | ForEach-Object {
            [System.Management.Automation.CompletionResult]::new($_.Name, $_.Name, 'ParameterName', $_.Description)
        }
        return
    }
    
    # Complete file paths
    Get-ChildItem -Path "$wordToComplete*" -ErrorAction SilentlyContinue | ForEach-Object {
        $path = if ($_.PSIsContainer) { "$($_.Name)/" } else { $_.Name }
        [System.Management.Automation.CompletionResult]::new($path, $path, 'ProviderItem', $_.FullName)
    }
}
`
	_, err := fmt.Fprint(w, script)
	return err
}
