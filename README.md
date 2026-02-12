# tftidy

A Go tool that recursively scans Terraform files and removes transient blocks: `moved`, `removed`, and `import`.

## Overview

`tftidy` unifies these tools into a single CLI:

- [`terraform-moved-remover`](https://github.com/mkusaka/terraform-moved-remover)
- [`terraform-removed-remover`](https://github.com/mkusaka/terraform-removed-remover)
- [`terraform-import-remover`](https://github.com/mkusaka/terraform-import-remover)

It helps clean up Terraform configurations after refactors and state operations by removing no-longer-needed transition blocks while preserving the rest of the file.

## Features

- Recursively scans directories for `.tf` files
- Removes selected block types: `moved`, `removed`, `import`
- Supports selecting block types with `--type`
- Supports dry-run mode (`--dry-run`)
- Preserves original file permissions on write
- Skips writes when no target blocks are found
- Applies HCL formatting after removal
- Optional whitespace normalization (`--normalize-whitespace`)
- Reports detailed processing statistics

## Requirements

- Go 1.24 or later

## Installation

### From Source

```bash
git clone https://github.com/mkusaka/tftidy.git
cd tftidy
go build -o tftidy ./cmd/tftidy
```

### Using Go Install

```bash
go install github.com/mkusaka/tftidy/cmd/tftidy@latest
```

This installs the binary into your `$GOPATH/bin` (or `$GOBIN` if set).

## Usage

```bash
tftidy [options] [directory]
```

If `directory` is not specified, the current directory is used.

### Options

- `-t, --type string`
  Comma-separated block types to remove.
  Valid values: `moved`, `removed`, `import`, `all`.
  Default: `moved,removed,import`
- `-n, --dry-run`
  Preview changes without modifying files.
- `-v, --verbose`
  Show each file being processed.
- `--normalize-whitespace`
  Normalize consecutive blank lines after removal.
- `--version`
  Show version.
- `-h, --help`
  Show help.

### Examples

Remove all supported block types in the current directory:

```bash
tftidy
```

Remove only `moved` blocks:

```bash
tftidy --type moved ./terraform
```

Remove `moved` and `import` blocks with whitespace normalization:

```bash
tftidy --type moved,import --normalize-whitespace ./terraform
```

Preview only (no file writes):

```bash
tftidy --dry-run ./terraform
```

## Migration from legacy tools

### Command migration paths

From [`terraform-moved-remover`](https://github.com/mkusaka/terraform-moved-remover):

```bash
terraform-moved-remover [options] [directory]
# ->
tftidy --type moved [options] [directory]
```

From [`terraform-removed-remover`](https://github.com/mkusaka/terraform-removed-remover):

```bash
terraform-removed-remover [options] [directory]
# ->
tftidy --type removed [options] [directory]
```

From [`terraform-import-remover`](https://github.com/mkusaka/terraform-import-remover):

```bash
terraform-import-remover [options] [directory]
# ->
tftidy --type import [options] [directory]
```

### Option mapping

- `-dry-run` / `--dry-run` -> `-n` / `--dry-run`
- `-verbose` / `--verbose` -> `-v` / `--verbose`
- `-normalize-whitespace` / `--normalize-whitespace` -> `--normalize-whitespace`
- `-help` / `--help` -> `-h` / `--help`
- `-version` / `--version` -> `--version`

### Behavioral differences

- Files are not rewritten when no target blocks are found (no format-only writes).
- Original file permissions are preserved when files are rewritten.
- Exit codes are strict:
  - `1` for runtime/file processing errors
  - `2` for usage/argument errors

## Example Output

```text
Files processed: 15
Files modified: 7
Files errored: 0

Blocks removed:
  moved:   12
  removed: 3
  import:  5
  total:   20
```

## Exit Codes

- `0`: success
- `1`: runtime/file processing error(s)
- `2`: usage/argument error

If processing errors occur, `tftidy` continues other files and returns `1` at the end.

## How It Works

`tftidy` parses Terraform files using HashiCorp HCL v2, locates target top-level blocks, removes their byte ranges, formats the result, and writes changes in place (unless dry-run).

File discovery is powered by `github.com/boyter/gocodewalker`, and excludes `.terraform` / `.terragrunt-cache` directories.

## Development

```bash
go build -v ./cmd/tftidy
go test -v ./...
go test -race ./...
```

## License

MIT
