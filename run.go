package tftidy

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

var Version = "dev"

var allowedBlockTypes = []string{"moved", "removed", "import"}

type stats struct {
	filesProcessed int
	filesModified  int
	filesErrored   int
	blockCounts    map[string]int
}

func Run(args []string, stdout, stderr io.Writer) int {
	return run(args, stdout, stderr)
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := pflag.NewFlagSet("tftidy", pflag.ContinueOnError)
	fs.SortFlags = false
	fs.SetOutput(stderr)

	rawTypes := fs.StringP("type", "t", "moved,removed,import", "Block types to remove, comma-separated")
	dryRun := fs.BoolP("dry-run", "n", false, "Preview changes without modifying files")
	verbose := fs.BoolP("verbose", "v", false, "Show each file being processed")
	normalizeWhitespace := fs.Bool("normalize-whitespace", false, "Normalize consecutive blank lines after removal")
	showVersion := fs.Bool("version", false, "Show version")
	showHelp := fs.BoolP("help", "h", false, "Show help")

	if err := fs.Parse(args); err != nil {
		writef(stderr, "Error: %v\n\n", err)
		printUsage(stderr)
		return 2
	}

	if *showHelp {
		printUsage(stdout)
		return 0
	}

	if *showVersion {
		writef(stdout, "tftidy %s\n", Version)
		return 0
	}

	remaining := fs.Args()
	if len(remaining) > 1 {
		writef(stderr, "Error: expected at most one directory argument\n\n")
		printUsage(stderr)
		return 2
	}

	blockTypes, err := parseBlockTypes(*rawTypes)
	if err != nil {
		writef(stderr, "Error: %v\n", err)
		return 2
	}

	dir := "."
	if len(remaining) == 1 {
		dir = remaining[0]
	}

	info, err := os.Stat(dir)
	if err != nil {
		writef(stderr, "Error: %v\n", err)
		return 1
	}
	if !info.IsDir() {
		writef(stderr, "Error: %s is not a directory\n", dir)
		return 1
	}

	files, err := discoverFiles(dir)
	if err != nil {
		writef(stderr, "Error: failed to discover Terraform files: %v\n", err)
		return 1
	}

	st := stats{blockCounts: make(map[string]int, len(blockTypes))}
	for _, blockType := range blockTypes {
		st.blockCounts[blockType] = 0
	}

	for _, path := range files {
		st.filesProcessed++
		if *verbose {
			writef(stdout, "Processing: %s\n", path)
		}

		fileInfo, err := os.Stat(path)
		if err != nil {
			recordFileError(stderr, path, err, &st)
			continue
		}

		content, err := os.ReadFile(path)
		if err != nil {
			recordFileError(stderr, path, err, &st)
			continue
		}

		updated, counts, err := removeBlocks(content, path, blockTypes)
		if err != nil {
			recordFileError(stderr, path, err, &st)
			continue
		}

		removedInFile := sumCounts(counts)
		if removedInFile == 0 {
			continue
		}

		if *dryRun {
			st.filesModified++
			addCounts(&st, counts)
			continue
		}

		if *normalizeWhitespace {
			updated = normalizeConsecutiveNewlines(updated)
		}

		if err := os.WriteFile(path, updated, fileInfo.Mode().Perm()); err != nil {
			recordFileError(stderr, path, err, &st)
			continue
		}

		st.filesModified++
		addCounts(&st, counts)
	}

	printStats(stdout, st, blockTypes)

	if st.filesErrored > 0 {
		return 1
	}

	return 0
}

func parseBlockTypes(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("--type must not be empty")
	}

	validSet := map[string]struct{}{
		"moved":   {},
		"removed": {},
		"import":  {},
	}

	result := make([]string, 0, len(allowedBlockTypes))
	seen := make(map[string]struct{}, len(allowedBlockTypes))

	appendUnique := func(blockType string) {
		if _, ok := seen[blockType]; ok {
			return
		}
		seen[blockType] = struct{}{}
		result = append(result, blockType)
	}

	parts := strings.Split(raw, ",")
	for _, part := range parts {
		blockType := strings.ToLower(strings.TrimSpace(part))
		if blockType == "" {
			continue
		}

		if blockType == "all" {
			for _, known := range allowedBlockTypes {
				appendUnique(known)
			}
			continue
		}

		if _, ok := validSet[blockType]; !ok {
			return nil, fmt.Errorf("unknown block type %q (valid: moved,removed,import,all)", blockType)
		}

		appendUnique(blockType)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("--type must include at least one block type")
	}

	return result, nil
}

func printUsage(w io.Writer) {
	writeln(w, "tftidy - Remove transient blocks (moved, removed, import) from Terraform files")
	writeln(w)
	writeln(w, "Usage: tftidy [options] [directory]")
	writeln(w)
	writeln(w, "Options:")
	writeln(w, "  -t, --type string              Block types to remove, comma-separated (default \"moved,removed,import\")")
	writeln(w, "  -n, --dry-run                  Preview changes without modifying files")
	writeln(w, "  -v, --verbose                  Show each file being processed")
	writeln(w, "      --normalize-whitespace     Normalize consecutive blank lines after removal")
	writeln(w, "      --version                  Show version")
	writeln(w, "  -h, --help                     Show help")
}

func recordFileError(stderr io.Writer, path string, err error, st *stats) {
	st.filesErrored++
	writef(stderr, "Error processing %s: %v\n", path, err)
}

func addCounts(st *stats, counts map[string]int) {
	for blockType, count := range counts {
		st.blockCounts[blockType] += count
	}
}

func sumCounts(counts map[string]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}

func printStats(stdout io.Writer, st stats, blockTypes []string) {
	writef(stdout, "Files processed: %d\n", st.filesProcessed)
	writef(stdout, "Files modified: %d\n", st.filesModified)
	writef(stdout, "Files errored: %d\n", st.filesErrored)
	writeln(stdout)
	writeln(stdout, "Blocks removed:")

	total := 0
	for _, blockType := range blockTypes {
		count := st.blockCounts[blockType]
		total += count
		writef(stdout, "  %-8s %d\n", blockType+":", count)
	}

	writef(stdout, "  %-8s %d\n", "total:", total)
}

func writef(w io.Writer, format string, args ...any) {
	_, _ = fmt.Fprintf(w, format, args...)
}

func writeln(w io.Writer, args ...any) {
	_, _ = fmt.Fprintln(w, args...)
}
