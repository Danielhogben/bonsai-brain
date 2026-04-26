# Basic Agent Example

A minimal demonstration of Bonsai Brain v3 showing:

- `QueryEngine` with tool registration
- 3-state permission pipeline (`allow` / `block` / `ask-user`)
- Input guardrails (blocked keywords, max length)
- Output middleware (truncation)
- Hierarchical Agent wrapper

## Run

```bash
go run main.go
```

## What to try

- Change the user message to contain `"password"` and watch the input guardrail block it.
- Remove the `eng.AskUser` callback and see the permission pipeline error out.
- Add more tools to the engine and observe the mock model calling them.
