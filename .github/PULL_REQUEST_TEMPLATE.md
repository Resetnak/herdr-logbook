<!-- Keep the diff focused. Link the issue it closes. -->

## What and why

<!-- What this changes and the reason. Closes #NNN. -->

## Verification

- [ ] `go test -race ./...`
- [ ] `go vet ./...`
- [ ] `staticcheck ./...`
- [ ] `govulncheck ./...`
- [ ] `go build -o bin/herdr-logbook ./cmd/herdr-logbook`

## Product boundaries

- [ ] No telemetry, network calls, cloud sync, AI, or automatic Git operations.
- [ ] No execution of commands found in notes; editor still resolved as argv, never via a shell.
- [ ] Markdown stays canonical; any cache remains disposable.
