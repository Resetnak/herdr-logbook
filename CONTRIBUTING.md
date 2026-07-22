# Contributing

Use Go from `go.mod`, keep behavior offline, and preserve Markdown as the source of truth. Changes should stay within the focused developer-memory scope described in `HERDR_LOGBOOK_AGENT_BRIEF.md`.

Before opening a pull request, run:

```bash
go test ./...
go test -race ./...
go vet ./...
staticcheck ./...
```

Add direct unit tests for domain behavior and Bubble Tea update tests for UI state. Use temporary directories for filesystem and Git integration tests. Do not add telemetry, AI APIs, cloud sync, SQLite, shell-interpolated editor commands, or automatic repository modifications.

Cross-platform changes must retain Linux, macOS, and Windows builds. Do not claim a real-host compatibility row without recording the actual test in `docs/herdr-compatibility.md`.
