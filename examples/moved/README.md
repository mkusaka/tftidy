# moved Examples

Example Terraform configurations for `moved` block cleanup.

## Run

```bash
./tftidy --type moved ./examples/moved
```

## Includes

- `main.tf`: basic `moved` examples
- `modules/`: nested module examples
- `edge_cases/empty.tf`: empty file
- `edge_cases/only_moved.tf`: file with only `moved` blocks
- `edge_cases/commented.tf`: commented `moved` blocks
