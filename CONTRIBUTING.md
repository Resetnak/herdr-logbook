# Contributing to Herdr Logbook

First off, thank you for considering contributing to **Herdr Logbook**! Contributions from the community help make this plugin better for everyone.

---

## 📜 Principles & Scope

Herdr Logbook is a local, offline, Markdown-first working memory plugin for Herdr. When contributing, please keep the following design goals in mind:

- **Markdown remains canonical**: Plain `.md` files are the source of truth. Indexes and caches are disposable.
- **Offline & Private**: Zero telemetry, no cloud sync, no AI APIs, no automatic Git commits, and no arbitrary shell execution from notes.
- **Bring Your Own Editor**: Full document editing is delegated to `$EDITOR`; the TUI does not implement a text editor.
- **Cross-Platform**: Support Linux, macOS, and Windows.

---

## 🛠️ Local Development & Testing

Before submitting a Pull Request, please ensure all verification checks pass locally:

```bash
# 1. Run unit & integration tests
go test ./...

# 2. Check for race conditions
go test -race ./...

# 3. Code analysis & linting
go vet ./...
```

For shell script modifications:
```bash
bash -n scripts/open.sh scripts/install.sh
```

---

## 📥 Submitting Pull Requests

1. **Fork the Repository** and create a feature branch (`git checkout -b feature/my-cool-feature`).
2. **Write Unit Tests** for any new logic or Bubble Tea UI state changes.
3. **Run Verification Commands** listed above to make sure everything passes cleanly.
4. **Commit with Clear Messages** explaining *why* the change was made.
5. **Add a `CHANGELOG.md` Entry** under `## Unreleased` for anything a user would notice: a fixed bug, a new flag, changed behaviour. Purely internal refactors and test-only changes do not need one.
6. **Open a Pull Request** describing your changes and testing performed.

---

## 📄 License

By contributing to Herdr Logbook, you agree that your contributions will be licensed under the project's [MIT License](LICENSE).
