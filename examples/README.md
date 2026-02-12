# tftidy Examples

This directory contains legacy-compatible example Terraform files migrated from:

- `terraform-moved-remover`
- `terraform-removed-remover`
- `terraform-import-remover`

## Directory Layout

```text
examples/
├── moved/      # examples focused on moved blocks
├── removed/    # examples focused on removed blocks
└── import/     # examples focused on import blocks
```

Each subdirectory includes:

- `main.tf`
- `modules/`
- `edge_cases/`

## Usage

Run all block cleanups against all examples:

```bash
./tftidy ./examples
```

Run type-specific cleanup for one set:

```bash
./tftidy --type moved ./examples/moved
./tftidy --type removed ./examples/removed
./tftidy --type import ./examples/import
```

Dry-run:

```bash
./tftidy --dry-run --verbose ./examples
```
