# Contributing to charm-table-cli

Thank you for your interest in contributing!

## How to Contribute

1. **Fork** the repository
2. **Create** a feature branch (`git checkout -b feat/my-feature`)
3. **Commit** your changes using [Conventional Commits](https://www.conventionalcommits.org/)
4. **Push** to your branch (`git push origin feat/my-feature`)
5. **Open** a Pull Request

## Development

```bash
# Clone
git clone https://github.com/junhinhow/charm-table-cli.git
cd charm-table-cli

# Build
go build -o charm-table-cli .

# Test
go test ./...

# Run
echo "a,b,c\n1,2,3" | ./charm-table-cli
```

## Commit Messages

Use conventional commits:
- `feat:` new feature
- `fix:` bug fix
- `refactor:` code refactoring
- `docs:` documentation
- `chore:` maintenance

## Code Style

- Run `go fmt` before committing
- Run `go vet` to check for issues
- Keep functions focused and well-named

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
