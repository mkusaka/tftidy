# removed Examples

Example Terraform configurations for `removed` block cleanup.

## Run

```bash
./tftidy --type removed ./examples/removed
```

## Includes

- `main.tf`: basic `removed` examples
- `modules/`: nested module examples
- `edge_cases/empty.tf`: empty file
- `edge_cases/only_removed.tf`: file with only `removed` blocks
- `edge_cases/commented.tf`: commented `removed` blocks
