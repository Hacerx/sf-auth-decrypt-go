package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/hacerx/sf-auth-decrypt-go/authdecrypt"
)

type runOptions struct {
	extraOptions []authdecrypt.Option
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, runOptions{}))
}

func run(args []string, stdout, stderr io.Writer, opts runOptions) int {
	flags := flag.NewFlagSet("sf-auth-decrypt-go", flag.ContinueOnError)
	flags.SetOutput(stderr)

	showSecrets := flags.Bool("show-secrets", false, "print the full decrypted org record, including sensitive values")
	homeDir := flags.String("home", "", "home directory containing .sfdx and .sf state")
	legacyStateDir := flags.String("legacy-state-dir", "", "explicit .sfdx state directory")
	modernStateDir := flags.String("modern-state-dir", "", "explicit .sf state directory")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: sf-auth-decrypt-go [flags] <alias-or-username>")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "By default, output is a safe summary. Use --show-secrets only when you intentionally need the full decrypted record.")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 1 {
		flags.Usage()
		return 2
	}

	clientOptions := make([]authdecrypt.Option, 0, 3+len(opts.extraOptions))
	if strings.TrimSpace(*homeDir) != "" {
		clientOptions = append(clientOptions, authdecrypt.WithHomeDir(*homeDir))
	}
	if strings.TrimSpace(*legacyStateDir) != "" {
		clientOptions = append(clientOptions, authdecrypt.WithLegacyStateDir(*legacyStateDir))
	}
	if strings.TrimSpace(*modernStateDir) != "" {
		clientOptions = append(clientOptions, authdecrypt.WithModernStateDir(*modernStateDir))
	}
	clientOptions = append(clientOptions, opts.extraOptions...)

	client, err := authdecrypt.New(clientOptions...)
	if err != nil {
		fmt.Fprintf(stderr, "failed to configure auth decrypt client: %v\n", err)
		return 1
	}

	record, err := client.ResolveOrg(context.Background(), flags.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "failed to resolve org: %v\n", err)
		return 1
	}

	output := any(summaryRecord(record))
	if *showSecrets {
		output = record
	}
	if err := writeJSON(stdout, output); err != nil {
		fmt.Fprintf(stderr, "failed to write output: %v\n", err)
		return 1
	}

	return 0
}

func writeJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func summaryRecord(record authdecrypt.OrgRecord) map[string]any {
	summary := map[string]any{
		"warning": "Sensitive fields are hidden. Re-run with --show-secrets only when you intend to handle decrypted org data.",
	}
	for _, field := range []string{"username", "userName", "orgId", "instanceUrl", "loginUrl", "isDevHub", "isScratchOrg", "isSandbox"} {
		if value, ok := record[field]; ok {
			summary[field] = value
		}
	}
	if fields := redactedFields(record); len(fields) > 0 {
		summary["redactedFields"] = fields
	}
	return summary
}

func redactedFields(record authdecrypt.OrgRecord) []string {
	fields := make([]string, 0)
	for field := range record {
		if isSensitiveField(field) {
			fields = append(fields, field)
		}
	}
	sort.Strings(fields)
	return fields
}

func isSensitiveField(field string) bool {
	lower := strings.ToLower(field)
	for _, marker := range []string{"token", "secret", "password", "key", "private", "session", "authorization"} {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}
