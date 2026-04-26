---
name: 🔌 Tool Integration
about: Propose or build a new tool for the ecosystem
title: '[TOOL] '
labels: tool, enhancement
assignees: ''
---

## Tool Name
What is the tool called?

## What It Does
Describe the functionality in 1-2 sentences.

## Target CLI / API
What does this wrap? (e.g. `docker`, `gh`, `curl`, a REST API)

## Example Usage
```go
// How would a developer use this tool?
engine.RegisterTool(
    tool.Schema{Name: "...", ...},
    myToolFunc,
)
```

## Prior Art
Are there similar tools in other frameworks? (LangChain, elizaOS, etc.)

## Want to Build This?
- [ ] Yes, I want to implement this!
- [ ] I need help getting started
- [ ] Just suggesting for now
