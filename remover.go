package tftidy

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type byteRange struct {
	start int
	end   int
}

func removeBlocks(content []byte, filename string, blockTypes []string, removeComments bool) ([]byte, map[string]int, error) {
	if removeComments {
		return removeBlocksWithComments(content, filename, blockTypes)
	}
	return removeBlocksPreservingComments(content, filename, blockTypes)
}

// removeBlocksWithComments uses hclwrite.RemoveBlock which naturally removes
// leading comments attached to the block (hclwrite stores them as child tokens).
func removeBlocksWithComments(content []byte, filename string, blockTypes []string) ([]byte, map[string]int, error) {
	file, diags := hclwrite.ParseConfig(content, filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, nil, fmt.Errorf("failed to parse %s: %s", filename, diags.Error())
	}

	typeSet := make(map[string]struct{}, len(blockTypes))
	for _, blockType := range blockTypes {
		typeSet[blockType] = struct{}{}
	}

	counts := make(map[string]int, len(blockTypes))
	body := file.Body()
	for _, block := range body.Blocks() {
		if _, ok := typeSet[block.Type()]; !ok {
			continue
		}
		body.RemoveBlock(block)
		counts[block.Type()]++
	}

	if sumCounts(counts) == 0 {
		return content, map[string]int{}, nil
	}

	return hclwrite.Format(file.Bytes()), counts, nil
}

// removeBlocksPreservingComments uses hclsyntax to get precise byte ranges
// that exclude leading comments, then removes blocks at the byte level.
func removeBlocksPreservingComments(content []byte, filename string, blockTypes []string) ([]byte, map[string]int, error) {
	syntaxFile, diags := hclsyntax.ParseConfig(content, filename, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, nil, fmt.Errorf("failed to parse %s: %s", filename, diags.Error())
	}

	syntaxBody, ok := syntaxFile.Body.(*hclsyntax.Body)
	if !ok {
		return nil, nil, fmt.Errorf("unexpected HCL body type in %s", filename)
	}

	typeSet := make(map[string]struct{}, len(blockTypes))
	for _, blockType := range blockTypes {
		typeSet[blockType] = struct{}{}
	}

	counts := make(map[string]int, len(blockTypes))
	ranges := make([]byteRange, 0)

	for _, block := range syntaxBody.Blocks {
		if _, ok := typeSet[block.Type]; !ok {
			continue
		}

		r := block.Range()
		ranges = append(ranges, byteRange{start: r.Start.Byte, end: r.End.Byte})
		counts[block.Type]++
	}

	if len(ranges) == 0 {
		return content, map[string]int{}, nil
	}

	result := append([]byte(nil), content...)
	for i := len(ranges) - 1; i >= 0; i-- {
		r := ranges[i]
		start := r.start
		end := r.end

		for start > 0 && (result[start-1] == ' ' || result[start-1] == '\t') {
			start--
		}

		for end < len(result) && (result[end] == '\r' || result[end] == '\n') {
			end++
			if result[end-1] == '\n' {
				break
			}
		}

		result = append(result[:start], result[end:]...)
	}

	return hclwrite.Format(result), counts, nil
}

func normalizeConsecutiveNewlines(content []byte) []byte {
	contentStr := string(content)
	replacer := strings.NewReplacer("\n\n\n", "\n\n", "\r\n\r\n\r\n", "\r\n\r\n")

	for {
		normalized := replacer.Replace(contentStr)
		if normalized == contentStr {
			break
		}
		contentStr = normalized
	}

	contentStr = strings.ReplaceAll(contentStr, "\r\n", "\n")
	contentStr = strings.TrimRight(contentStr, "\n") + "\n"

	if bytes.Contains(content, []byte("\r\n")) {
		contentStr = strings.ReplaceAll(contentStr, "\n", "\r\n")
	}

	return []byte(contentStr)
}
