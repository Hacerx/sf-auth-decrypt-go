package keychain

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
)

var errCredentialProgramMissing = errors.New("credential program missing")

const (
	linuxSecretToolInvalidSecretMessage = "invalid or unencryptable secret"
	linuxSecretToolInvalidSecretRetries = 3
)

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, program string, args ...string) (CommandResult, error) {
	cmd := exec.CommandContext(ctx, program, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	result := CommandResult{Stdout: stdout.String(), Stderr: stderr.String()}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
			return result, nil
		}
		return result, err
	}
	return result, nil
}

type CommandProvider struct {
	Program       string
	Args          []string
	Runner        CommandRunner
	ParsePassword func(CommandResult) (string, error)
	ShouldRetry   func(CommandResult) bool
	MaxRetries    int
}

func (p CommandProvider) Key(ctx context.Context, service, account string) ([]byte, error) {
	if err := ctxErr(ctx); err != nil {
		return nil, err
	}
	runner := p.Runner
	if runner == nil {
		runner = ExecRunner{}
	}
	for attempt := 0; ; attempt++ {
		if err := ctxErr(ctx); err != nil {
			return nil, err
		}
		result, err := runner.Run(ctx, p.Program, p.Args...)
		if err != nil {
			if isExecutableNotFound(err) {
				return nil, fmt.Errorf("%w: %w", ErrKeychain, errCredentialProgramMissing)
			}
			return nil, fmt.Errorf("%w", ErrKeychain)
		}
		if p.ShouldRetry != nil && p.ShouldRetry(result) && attempt < p.MaxRetries {
			continue
		}
		password, err := p.ParsePassword(result)
		if err != nil {
			return nil, err
		}
		return []byte(password), nil
	}
}

func NewLinuxSecretToolProvider(runner CommandRunner) CommandProvider {
	return CommandProvider{
		Program:       linuxSecretToolPath(),
		Args:          []string{"lookup", "user", AccountLocal, "domain", ServiceSFDX},
		Runner:        runner,
		ParsePassword: parseLinuxPassword,
		ShouldRetry:   isLinuxSecretToolTransientInvalidSecret,
		MaxRetries:    linuxSecretToolInvalidSecretRetries,
	}
}

func NewLinuxProvider(genericStateDir string, runner CommandRunner) Provider {
	return fallbackOnMissingProgramProvider{
		primary:  NewLinuxSecretToolProvider(runner),
		fallback: NewGenericFileProvider(genericStateDir),
	}
}

func NewDarwinSecurityProvider(runner CommandRunner) CommandProvider {
	return CommandProvider{
		Program:       "/usr/bin/security",
		Args:          []string{"find-generic-password", "-a", AccountLocal, "-s", ServiceSFDX, "-g"},
		Runner:        runner,
		ParsePassword: parseDarwinPassword,
	}
}

func NewPlatformProvider(genericStateDir string, runner CommandRunner) Provider {
	if useGenericUnixKeychain() || runtime.GOOS == "windows" {
		return NewGenericFileProvider(genericStateDir)
	}
	switch runtime.GOOS {
	case "darwin":
		return NewDarwinSecurityProvider(runner)
	case "linux":
		return NewLinuxProvider(genericStateDir, runner)
	default:
		return NewGenericFileProvider(genericStateDir)
	}
}

type fallbackOnMissingProgramProvider struct {
	primary  Provider
	fallback Provider
}

func (p fallbackOnMissingProgramProvider) Key(ctx context.Context, service, account string) ([]byte, error) {
	key, err := p.primary.Key(ctx, service, account)
	if err == nil {
		return key, nil
	}
	if errors.Is(err, errCredentialProgramMissing) {
		return p.fallback.Key(ctx, service, account)
	}
	return nil, err
}

func parseLinuxPassword(result CommandResult) (string, error) {
	if isLinuxSecretToolTransientInvalidSecret(result) {
		return "", fmt.Errorf("%w", ErrKeychain)
	}
	if result.ExitCode == 1 {
		return "", ErrMissingKey
	}
	if result.ExitCode != 0 {
		return "", fmt.Errorf("%w", ErrKeychain)
	}
	password := strings.TrimSpace(result.Stdout)
	if password == "" {
		return "", ErrMissingKey
	}
	return password, nil
}

func isLinuxSecretToolTransientInvalidSecret(result CommandResult) bool {
	return result.ExitCode == 1 && strings.Contains(result.Stderr, linuxSecretToolInvalidSecretMessage)
}

var securityPasswordPattern = regexp.MustCompile(`password:\s*"(.*)"`)

func parseDarwinPassword(result CommandResult) (string, error) {
	if result.ExitCode != 0 {
		return "", ErrMissingKey
	}
	match := securityPasswordPattern.FindStringSubmatch(result.Stderr)
	if len(match) != 2 || match[1] == "" {
		return "", ErrMissingKey
	}
	return match[1], nil
}

func useGenericUnixKeychain() bool {
	return strings.EqualFold(strings.TrimSpace(getenv("SF_USE_GENERIC_UNIX_KEYCHAIN")), "true") || strings.EqualFold(strings.TrimSpace(getenv("USE_GENERIC_UNIX_KEYCHAIN")), "true")
}

func linuxSecretToolPath() string {
	if program := strings.TrimSpace(getenv("SFDX_SECRET_TOOL_PATH")); program != "" {
		return program
	}
	return "/usr/bin/secret-tool"
}

func isExecutableNotFound(err error) bool {
	return errors.Is(err, exec.ErrNotFound) || errors.Is(err, fs.ErrNotExist) || errors.Is(err, os.ErrNotExist)
}

var getenv = os.Getenv
