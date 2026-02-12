package tftidy

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	t.Parallel()

	oldVersion := Version
	Version = "test-version"
	defer func() {
		Version = oldVersion
	}()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--version"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "tftidy test-version") {
		t.Fatalf("unexpected version output: %s", stdout.String())
	}
}

func TestRunHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Usage: tftidy [options] [directory]") {
		t.Fatalf("help output is missing usage: %s", stdout.String())
	}
}

func TestRunInvalidType(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--type", "unknown"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout, got: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "unknown block type") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunNonExistentDirectory(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"/definitely/not/found"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout, got: %s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Error:") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunTooManyArgs(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"one", "two"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "expected at most one directory argument") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestParseBlockTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{name: "default all explicit", raw: "moved,removed,import", want: []string{"moved", "removed", "import"}},
		{name: "all keyword", raw: "all", want: []string{"moved", "removed", "import"}},
		{name: "dedupe", raw: "moved, moved,import", want: []string{"moved", "import"}},
		{name: "mixed all dedupe", raw: "removed,all", want: []string{"removed", "moved", "import"}},
		{name: "unknown", raw: "moved,foo", wantErr: true},
		{name: "empty", raw: "", wantErr: true},
		{name: "comma only", raw: " , , ", wantErr: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseBlockTypes(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("unexpected parsed block types, expected %v got %v", tc.want, got)
			}
		})
	}
}

func TestRunNotDirectory(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "main.tf")
	if err := os.WriteFile(file, []byte("resource \"null_resource\" \"x\" {}\n"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{file}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "is not a directory") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}
