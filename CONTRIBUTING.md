Thanks for contributing! A few repo-specific notes to make contributions straightforward.

Repository layout
- This workspace contains multiple independent projects at the top level. Work is project-scoped: cd into the subproject before running build/test commands.

Project tooling preferences
- Python: prefer 'uv' for dependency management where configured. Use 'pytest' for tests, 'ruff' for lint, and 'mypy' for type checks.
- Node: prefer 'pnpm' over 'npm' when available. Use 'npm test' or 'pnpm test' to run tests.
- Kotlin/Java: use './gradlew' for builds and tests. Target Java 21+ where required.
- Go: use 'make' targets if provided (e.g., make build).

How to test changes
- Run the project's test suite and a single test locally (see copilot-instructions.md for per-ecosystem single-test commands).
- Ensure linters pass before opening a PR.

Pull request expectations
- Explain the change, link related issues, include testing steps, and run linters.
- Keep commits focused and squashed when appropriate.

Contact
- If unsure which subproject to modify, ask in an issue first.
