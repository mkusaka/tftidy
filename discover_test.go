package tftidy

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestDiscoverFilesSkipsCacheDirsAndFiltersTerraformFiles(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	mustMkdirAll(t, filepath.Join(tempDir, "nested"))
	mustMkdirAll(t, filepath.Join(tempDir, ".terraform"))
	mustMkdirAll(t, filepath.Join(tempDir, ".terragrunt-cache"))

	mustWriteFile(t, filepath.Join(tempDir, "main.tf"), "resource \"null_resource\" \"main\" {}\n", 0o644)
	mustWriteFile(t, filepath.Join(tempDir, "nested", "module.tf"), "resource \"null_resource\" \"nested\" {}\n", 0o644)
	mustWriteFile(t, filepath.Join(tempDir, "nested", "variables.tfvars"), "k = \"v\"\n", 0o644)
	mustWriteFile(t, filepath.Join(tempDir, ".terraform", "ignored.tf"), "resource \"null_resource\" \"ignored\" {}\n", 0o644)
	mustWriteFile(t, filepath.Join(tempDir, ".terragrunt-cache", "ignored.tf"), "resource \"null_resource\" \"ignored\" {}\n", 0o644)

	files, err := discoverFiles(tempDir)
	if err != nil {
		t.Fatalf("discoverFiles failed: %v", err)
	}

	expected := []string{
		filepath.Join(tempDir, "main.tf"),
		filepath.Join(tempDir, "nested", "module.tf"),
	}
	if !reflect.DeepEqual(files, expected) {
		t.Fatalf("unexpected files\nexpected: %#v\nactual: %#v", expected, files)
	}
}

func TestDiscoverFilesRespectsGitignore(t *testing.T) {
	t.Parallel()

	tempDir := t.TempDir()

	mustWriteFile(t, filepath.Join(tempDir, ".gitignore"), "ignored.tf\nnested/ignored.tf\n", 0o644)
	mustWriteFile(t, filepath.Join(tempDir, "kept.tf"), "resource \"null_resource\" \"kept\" {}\n", 0o644)
	mustWriteFile(t, filepath.Join(tempDir, "ignored.tf"), "resource \"null_resource\" \"ignored\" {}\n", 0o644)
	mustMkdirAll(t, filepath.Join(tempDir, "nested"))
	mustWriteFile(t, filepath.Join(tempDir, "nested", "kept.tf"), "resource \"null_resource\" \"kept_nested\" {}\n", 0o644)
	mustWriteFile(t, filepath.Join(tempDir, "nested", "ignored.tf"), "resource \"null_resource\" \"ignored_nested\" {}\n", 0o644)

	files, err := discoverFiles(tempDir)
	if err != nil {
		t.Fatalf("discoverFiles failed: %v", err)
	}

	expected := []string{
		filepath.Join(tempDir, "kept.tf"),
		filepath.Join(tempDir, "nested", "kept.tf"),
	}
	sort.Strings(expected)
	if !reflect.DeepEqual(files, expected) {
		t.Fatalf("unexpected files\nexpected: %#v\nactual: %#v", expected, files)
	}
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s failed: %v", dir, err)
	}
}

func mustWriteFile(t *testing.T, path, content string, perm os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatalf("write %s failed: %v", path, err)
	}
}
