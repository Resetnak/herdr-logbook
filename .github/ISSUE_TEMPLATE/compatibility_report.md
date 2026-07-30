---
name: Platform compatibility report
about: Report a real Herdr Logbook host verification
title: "[Compatibility]: "
labels: compatibility
assignees: ""
---

## Environment

- **Herdr Logbook version:**
- **Herdr version:**
- **Operating system and version:**
- **CPU architecture:**
- **Shell or PowerShell version:**
- **Installation method:** plugin manager / install script / source build
- **Filesystem used:** native / WSL filesystem / mounted Windows path / other

## Checklist

Mark only steps you actually exercised:

- [ ] Installed or updated Herdr Logbook successfully
- [ ] Opened the Hub from a real Herdr action
- [ ] Opened and closed the Hub repeatedly without stacking panes
- [ ] Captured a project note
- [ ] Created a decision
- [ ] Set and changed the active task
- [ ] Searched across notes
- [ ] Opened a note in the configured external editor
- [ ] Tested a repository path containing spaces
- [ ] Tested a repository path containing Unicode

## Result

Describe what worked, what failed, and the exact action that produced the
failure.

## Diagnostics

Run `herdr-logbook doctor --json`, inspect the output, and remove anything you
do not want to post before pasting it below.

Do not include note contents, selected terminal text, Git credentials, or
unsanitized remotes.

```json

```

## Additional logs

Include only logs needed to reproduce the problem. Redact personal paths or
other private data if needed.
