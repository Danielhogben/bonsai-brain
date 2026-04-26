# Contributing to Bonsai Brain

Thank you for considering a contribution! 🌳

## Quick Start

1. **Fork** the repo
2. **Clone** your fork
3. **Create a branch** (`git checkout -b feature/my-thing`)
4. **Make changes**
5. **Test** (`go test ./...` && `go build ./...`)
6. **Commit** (`git commit -m "feat: add my thing"`)
7. **Push** (`git push origin feature/my-thing`)
8. **Open a PR**

## Development Setup

```bash
go version  # needs 1.22+
go test ./...
go build ./...
```

## Code Style

- Go standard formatting (`gofmt`)
- Clear, readable code over clever code
- Every exported type/function needs a doc comment
- Keep functions under 50 lines when possible
- Prefer composition over inheritance

## Testing

- All new packages should have tests
- Run `go test ./...` before committing
- If you add a tool, add a test that validates the schema

## Commit Messages

Follow conventional commits:
- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation
- `test:` tests
- `refactor:` code restructuring
- `perf:` performance improvement
- `chore:` maintenance

## What to Contribute

See [ROADMAP.md](ROADMAP.md) for milestones and [DREAMBOARD.md](DREAMBOARD.md) for the vision.

**High-priority areas:**
- New tools (wrap your favorite CLI!)
- Provider integrations (Together AI, Fireworks, Cerebras)
- TUI components (Bubble Tea)
- Documentation and examples
- Benchmarks and performance optimization

## Getting Help

- Open an issue with the `question` label
- Comment on an existing issue
- Check the examples in `examples/`

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
