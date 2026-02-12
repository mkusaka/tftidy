package tftidy

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIntegrationRunRemovesAllTargetBlocks(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "main.tf")
	input := `resource "aws_instance" "main" {
  ami           = "ami-123456"
  instance_type = "t3.micro"
}

moved {
  from = aws_instance.old
  to   = aws_instance.main
}

removed {
  from = aws_instance.legacy
  lifecycle {
    destroy = false
  }
}

import {
  to = aws_instance.main
  id = "i-123456"
}
`
	if err := os.WriteFile(file, []byte(input), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{tempDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	result := string(content)
	if containsBlockDeclaration(result, "moved") || containsBlockDeclaration(result, "removed") || containsBlockDeclaration(result, "import") {
		t.Fatalf("target blocks should be removed:\n%s", result)
	}

	out := stdout.String()
	if !strings.Contains(out, "moved:") || !strings.Contains(out, "removed:") || !strings.Contains(out, "import:") {
		t.Fatalf("stats output is missing block counts: %s", out)
	}
}

func TestIntegrationRunDryRun(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "main.tf")
	input := `resource "aws_instance" "main" {
  ami = "ami-123456"
}

moved {
  from = aws_instance.old
  to   = aws_instance.main
}
`
	if err := os.WriteFile(file, []byte(input), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--dry-run", tempDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != input {
		t.Fatalf("dry-run must not modify files")
	}
	if !strings.Contains(stdout.String(), "moved:") {
		t.Fatalf("dry-run should still print counts: %s", stdout.String())
	}
}

func TestIntegrationRunSelectiveRemoval(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "main.tf")
	input := `moved {
  from = aws_instance.old
  to   = aws_instance.main
}

removed {
  from = aws_instance.legacy
  lifecycle {
    destroy = false
  }
}

import {
  to = aws_instance.main
  id = "i-123456"
}
`
	if err := os.WriteFile(file, []byte(input), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--type", "moved", tempDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	result := string(content)
	if containsBlockDeclaration(result, "moved") {
		t.Fatalf("moved block should be removed:\n%s", result)
	}
	if !containsBlockDeclaration(result, "removed") || !containsBlockDeclaration(result, "import") {
		t.Fatalf("non-target blocks should remain:\n%s", result)
	}
}

func TestIntegrationRunPreservesPermissions(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "main.tf")
	input := `moved {
  from = aws_instance.old
  to   = aws_instance.main
}
`
	if err := os.WriteFile(file, []byte(input), 0o600); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	before, err := os.Stat(file)
	if err != nil {
		t.Fatalf("stat before failed: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{tempDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}

	after, err := os.Stat(file)
	if err != nil {
		t.Fatalf("stat after failed: %v", err)
	}
	if before.Mode().Perm() != after.Mode().Perm() {
		t.Fatalf("permissions changed: before=%#o after=%#o", before.Mode().Perm(), after.Mode().Perm())
	}
}

func TestIntegrationRunNoWriteWhenNoTargetBlocks(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()
	file := filepath.Join(tempDir, "main.tf")
	input := "resource \"aws_instance\" \"main\" {\nami=\"ami-123456\"\n}\n"
	if err := os.WriteFile(file, []byte(input), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	beforeInfo, err := os.Stat(file)
	if err != nil {
		t.Fatalf("stat before failed: %v", err)
	}
	beforeModTime := beforeInfo.ModTime()

	time.Sleep(20 * time.Millisecond)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{tempDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d stderr=%s", code, stderr.String())
	}

	afterInfo, err := os.Stat(file)
	if err != nil {
		t.Fatalf("stat after failed: %v", err)
	}
	if !afterInfo.ModTime().Equal(beforeModTime) {
		t.Fatalf("file should not be rewritten when no target block exists")
	}

	content, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if string(content) != input {
		t.Fatalf("content should remain unchanged when no target block exists")
	}
}
