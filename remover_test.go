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

			output, counts, err := removeBlocks([]byte(input), "main.tf", []string{tc.blockType})
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

	output, counts, err := removeBlocks([]byte(input), "main.tf", []string{"moved", "import"})
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

	output, counts, err := removeBlocks(input, "main.tf", []string{"moved"})
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

	output, _, err := removeBlocks([]byte(input), "main.tf", []string{"moved"})
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

	_, _, err := removeBlocks([]byte("this is not valid HCL"), "main.tf", []string{"moved"})
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

func containsBlockDeclaration(content, blockType string) bool {
	trimmed := strings.TrimSpace(content)
	if strings.HasPrefix(trimmed, blockType+" {") {
		return true
	}
	return strings.Contains(content, "\n"+blockType+" {")
}
