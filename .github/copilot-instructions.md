Purpose

This file guides Copilot/AI coding sessions for this multi-project workspace. It consolidates build/test/lint commands, high-level architecture, and repository-specific conventions so an AI assistant can act with minimal hand-holding.

Quick repo snapshot

- Multi-project workspace (working dir: /home/donn). Active subprojects include: deer-flow (Python/TS), potpie (Python), zeroclaw (Go), custom-mmo (Kotlin/Gradle), router-tool (Node/Puppeteer), osrs-launcher (Java). Treat each subproject as an independent build unit — run commands from the project's folder.

Build / test / lint (per ecosystem)

Python
- Install deps: pip install -r requirements.txt  OR use uv where present (uv sync).
- Run full tests: pytest
- Run a single test: pytest path/to/test_file.py::test_function_name
- Lint: ruff check .
- Type-check: mypy .

Node / frontend
- Install: pnpm install (preferred) or npm install
- Run tests: npm test (or pnpm test)
- Run a single Jest test: npm test -- -t "test name"
- Lint: npm run lint  (or pnpm lint)

Go
- Build: make build (zeroclaw)
- Run a single test: go test ./pkg/path -run TestName

Kotlin / Java (Gradle)
- Build: ./gradlew build
- Run tests (module): ./gradlew :module:test
- Run a single JUnit test: ./gradlew :module:test --tests "com.example.MyTest.testMethod"
- Run server: ./gradlew :server:run
- Single-file Java compile (osrs-launcher): javac RSModClient.java; run with java RSModClient <args>

High-level architecture

- Workspace is composed of many independent projects. Each project has its own tooling and README. Prefer cd into the specific project before running build/test commands.
- AI/Agent infra: OpenClaw is the primary agent gateway (systemd user service). Local LLM inference uses Ollama (LAN:192.168.0.245:11434) and Unsloth for GPU tasks.
- Game stack: custom-mmo is a Gradle multi-module project (protocol <- core <- server/client). RSMod (external host) is a Kotlin/Gradle project; clients in this workspace target it.
- Utilities: router-tool (Puppeteer), osrs-launcher (single-file Java clients), rsps-dev (cache tools).

Key conventions

- Preferred package managers:
  - Python: use uv where configured (uv sync).
  - Node: prefer pnpm over npm.
- Linters & type tools: Python uses ruff + mypy; JS/TS uses ESLint + Prettier.
- Java/Kotlin: target Java 21+ for Gradle projects when specified.
- PATH: ensure ~/go/bin and ~/.local/bin are in PATH for OpenClaw companion binaries.
- Port conflicts: RSMod / RSBox / OpenRSC share port 43594 — stop one before starting another.

AI assistant integration files

- Consult CLAUDE.md, AGENTS.md, GEMINI.md, AGENT_DEPLOYMENT.md for project-specific operational notes before performing elevated actions (starting services, modifying systemd services, etc.).

Where to look next

- Start with top-level README.md, then open per-project README or CLAUDE.md for detailed run instructions.
