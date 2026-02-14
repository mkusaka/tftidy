package tftidy

import (
	"bytes"
	"strings"
	"testing"
)

func TestRemoveBlocksSingleTypes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		blockType string
		block     string
	}{
		{
			name:      "moved",
			blockType: "moved",
			block: `moved {
  from = aws_instance.old
  to   = aws_instance.main
}`,
		},
		{
			name:      "removed",
			blockType: "removed",
			block: `removed {
  from = aws_instance.old
  lifecycle {
    destroy = false
  }
}`,
		},
		{
			name:      "import",
			blockType: "import",
			block: `import {
  to = aws_instance.main
  id = "i-123456"
}`,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			input := `resource "aws_instance" "main" {
  ami           = "ami-123456"
  instance_type = "t3.micro"
}

` + tc.block + `

resource "aws_s3_bucket" "data" {
  bucket = "example-bucket"
}
`

			output, counts, err := removeBlocks([]byte(input), "main.tf", []string{tc.blockType}, false)
			if err != nil {
				t.Fatalf("removeBlocks failed: %v", err)
			}

			if got := counts[tc.blockType]; got != 1 {
				t.Fatalf("expected one %s block removed, got %d", tc.blockType, got)
			}

			outputStr := string(output)
			if containsBlockDeclaration(outputStr, tc.blockType) {
				t.Fatalf("output still contains %s block:\n%s", tc.blockType, outputStr)
			}

			if !strings.Contains(outputStr, `resource "aws_instance" "main"`) {
				t.Fatalf("expected main resource to remain:\n%s", outputStr)
			}
		})
	}
}

func TestRemoveBlocksSelectiveTypes(t *testing.T) {
	t.Parallel()

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
  id = "i-abcdef"
}
`

	output, counts, err := removeBlocks([]byte(input), "main.tf", []string{"moved", "import"}, false)
	if err != nil {
		t.Fatalf("removeBlocks failed: %v", err)
	}

	if counts["moved"] != 1 || counts["import"] != 1 {
		t.Fatalf("unexpected counts: %#v", counts)
	}
	if _, ok := counts["removed"]; ok {
		t.Fatalf("removed block should not be counted when not targeted: %#v", counts)
	}

	outputStr := string(output)
	if containsBlockDeclaration(outputStr, "moved") {
		t.Fatalf("moved block should be removed:\n%s", outputStr)
	}
	if containsBlockDeclaration(outputStr, "import") {
		t.Fatalf("import block should be removed:\n%s", outputStr)
	}
	if !containsBlockDeclaration(outputStr, "removed") {
		t.Fatalf("removed block should remain:\n%s", outputStr)
	}
}

func TestRemoveBlocksNoMatchReturnsOriginal(t *testing.T) {
	t.Parallel()

	input := []byte(`resource "aws_instance" "main" {
ami = "ami-123456"
}
`)

	output, counts, err := removeBlocks(input, "main.tf", []string{"moved"}, false)
	if err != nil {
		t.Fatalf("removeBlocks failed: %v", err)
	}
	if len(counts) != 0 {
		t.Fatalf("expected empty counts for no match, got %#v", counts)
	}
	if !bytes.Equal(output, input) {
		t.Fatalf("content should be unchanged when no blocks matched\nexpected:\n%s\nactual:\n%s", string(input), string(output))
	}
}

func TestRemoveBlocksPreservesLeadingComments(t *testing.T) {
	t.Parallel()

	input := `resource "aws_instance" "main" {
  ami = "ami-123456"
}

# This comment should remain
moved {
  from = aws_instance.old
  to   = aws_instance.main
}
`

	output, _, err := removeBlocks([]byte(input), "main.tf", []string{"moved"}, false)
	if err != nil {
		t.Fatalf("removeBlocks failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "# This comment should remain") {
		t.Fatalf("leading comment was removed:\n%s", outputStr)
	}
	if containsBlockDeclaration(outputStr, "moved") {
		t.Fatalf("moved block should be removed:\n%s", outputStr)
	}
}

func TestRemoveBlocksInvalidHCL(t *testing.T) {
	t.Parallel()

	_, _, err := removeBlocks([]byte("this is not valid HCL"), "main.tf", []string{"moved"}, false)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
}

func TestNormalizeConsecutiveNewlines(t *testing.T) {
	t.Parallel()

	input := []byte("a\n\n\n\n\nb\n\n\n")
	got := normalizeConsecutiveNewlines(input)

	if string(got) != "a\n\nb\n" {
		t.Fatalf("unexpected normalized output: %q", string(got))
	}
}

func TestNormalizeConsecutiveNewlinesCRLF(t *testing.T) {
	t.Parallel()

	input := []byte("a\r\n\r\n\r\nb\r\n\r\n")
	got := normalizeConsecutiveNewlines(input)

	if string(got) != "a\r\n\r\nb\r\n" {
		t.Fatalf("unexpected normalized output: %q", string(got))
	}
}

func TestRemoveBlocksOnlyTargetBlocks(t *testing.T) {
	t.Parallel()

	input := `moved {
  from = aws_instance.old
  to   = aws_instance.new
}

import {
  to = aws_instance.new
  id = "i-abc"
}
`

	output, counts, err := removeBlocks([]byte(input), "main.tf", []string{"moved", "import"}, false)
	if err != nil {
		t.Fatalf("removeBlocks failed: %v", err)
	}

	if counts["moved"] != 1 || counts["import"] != 1 {
		t.Fatalf("unexpected counts: %#v", counts)
	}
	if strings.TrimSpace(string(output)) != "" {
		t.Fatalf("expected file to become empty/whitespace only, got:\n%s", string(output))
	}
}

func TestRemoveBlocksCommentedBlocksRemain(t *testing.T) {
	t.Parallel()

	input := `resource "aws_instance" "main" {
  ami = "ami-123456"
}

# moved {
#   from = aws_instance.old
#   to   = aws_instance.main
# }
`

	output, counts, err := removeBlocks([]byte(input), "main.tf", []string{"moved"}, false)
	if err != nil {
		t.Fatalf("removeBlocks failed: %v", err)
	}

	if len(counts) != 0 {
		t.Fatalf("expected no removals for commented block, got %#v", counts)
	}
	if !bytes.Equal(output, []byte(input)) {
		t.Fatalf("commented block should remain untouched")
	}
}

func TestRemoveBlocksTrailingTargetBlockWithNormalize(t *testing.T) {
	t.Parallel()

	input := `module "hoge" {
  source = "fuga"
}

moved {
  from = module.old_hoge
  to   = module.hoge
}
`

	output, counts, err := removeBlocks([]byte(input), "main.tf", []string{"moved"}, false)
	if err != nil {
		t.Fatalf("removeBlocks failed: %v", err)
	}
	if counts["moved"] != 1 {
		t.Fatalf("expected one moved removal, got %#v", counts)
	}

	normalized := normalizeConsecutiveNewlines(output)
	expected := `module "hoge" {
  source = "fuga"
}
`
	if string(normalized) != expected {
		t.Fatalf("unexpected normalized content\nexpected:\n%s\nactual:\n%s", expected, string(normalized))
	}
}

func TestRemoveBlocksWithRemoveComments(t *testing.T) {
	t.Parallel()

	input := `resource "aws_instance" "main" {
  ami = "ami-123456"
}

# This comment describes the moved block
moved {
  from = aws_instance.old
  to   = aws_instance.main
}
`

	output, counts, err := removeBlocks([]byte(input), "main.tf", []string{"moved"}, true)
	if err != nil {
		t.Fatalf("removeBlocks failed: %v", err)
	}
	if counts["moved"] != 1 {
		t.Fatalf("expected one moved removal, got %#v", counts)
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "# This comment describes the moved block") {
		t.Fatalf("leading comment should be removed with --remove-comments:\n%s", outputStr)
	}
	if !strings.Contains(outputStr, `resource "aws_instance" "main"`) {
		t.Fatalf("non-target resource should remain:\n%s", outputStr)
	}
}

func TestRemoveBlocksWithRemoveCommentsMultipleLines(t *testing.T) {
	t.Parallel()

	input := `resource "aws_instance" "main" {
  ami = "ami-123456"
}

# Comment line 1
# Comment line 2
# Comment line 3
moved {
  from = aws_instance.old
  to   = aws_instance.main
}
`

	output, _, err := removeBlocks([]byte(input), "main.tf", []string{"moved"}, true)
	if err != nil {
		t.Fatalf("removeBlocks failed: %v", err)
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "# Comment line") {
		t.Fatalf("all leading comment lines should be removed:\n%s", outputStr)
	}
}

func TestRemoveBlocksWithRemoveCommentsBlankLineBreaksAssociation(t *testing.T) {
	t.Parallel()

	input := `resource "aws_instance" "main" {
  ami = "ami-123456"
}

# This comment is not associated

moved {
  from = aws_instance.old
  to   = aws_instance.main
}
`

	output, _, err := removeBlocks([]byte(input), "main.tf", []string{"moved"}, true)
	if err != nil {
		t.Fatalf("removeBlocks failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "# This comment is not associated") {
		t.Fatalf("comment separated by blank line should remain:\n%s", outputStr)
	}
}

func TestRemoveBlocksWithRemoveCommentsDoubleSlash(t *testing.T) {
	t.Parallel()

	input := `resource "aws_instance" "main" {
  ami = "ami-123456"
}

// This is a double-slash comment
moved {
  from = aws_instance.old
  to   = aws_instance.main
}
`

	output, _, err := removeBlocks([]byte(input), "main.tf", []string{"moved"}, true)
	if err != nil {
		t.Fatalf("removeBlocks failed: %v", err)
	}

	outputStr := string(output)
	if strings.Contains(outputStr, "// This is a double-slash comment") {
		t.Fatalf("double-slash comment should be removed:\n%s", outputStr)
	}
}

func TestRemoveBlocksWithRemoveCommentsFalsePreservesComments(t *testing.T) {
	t.Parallel()

	input := `resource "aws_instance" "main" {
  ami = "ami-123456"
}

# This comment should remain
moved {
  from = aws_instance.old
  to   = aws_instance.main
}
`

	output, _, err := removeBlocks([]byte(input), "main.tf", []string{"moved"}, false)
	if err != nil {
		t.Fatalf("removeBlocks failed: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "# This comment should remain") {
		t.Fatalf("comment should be preserved when removeComments=false:\n%s", outputStr)
	}
}

func containsBlockDeclaration(content, blockType string) bool {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, blockType+" {") {
		return true
	}
	return strings.Contains(content, "\n"+blockType+" {")
}
