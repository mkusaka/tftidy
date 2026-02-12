# import Examples

Example Terraform configurations for `import` block cleanup.

## Run

```bash
./tftidy --type import ./examples/import
```

## Includes

- `main.tf`: basic `import` examples
- `modules/`: nested module examples
- `edge_cases/empty.tf`: empty file
- `edge_cases/only_import.tf`: file with only `import` blocks
- `edge_cases/commented.tf`: commented `import` blocks
