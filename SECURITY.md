# Security

Herdr Logbook is offline at runtime and stores canonical user data as Markdown. Please report a suspected vulnerability privately through GitHub Security Advisories rather than a public issue. Include the affected version, platform, reproduction steps, and whether a crafted repository, note, remote URL, or environment variable is required.

Install scripts are the only networked component. They download a version-pinned release asset from this repository and verify its SHA-256 checksum before extraction. The application does not execute note content, interpolate titles through a shell, emit selected terminal content in diagnostics, or persist credentials from Git remotes.

## Trust boundaries

A cloned repository is untrusted input. Its `.herdr-logbook.toml` can pin the project identity, display name, and the store location *within* the repository — it cannot choose where notes are stored. Repo-local storage stays an explicit opt-in you make through your own config or `--storage`, so a repository cannot redirect your working memory into its own worktree and have it pushed to its remote.

Note text is untrusted too, because it can arrive from a pipe (`capture --stdin`) or from a file another tool wrote. Captures and `now` tasks reject terminal control characters, and titles and previews strip them, so a stored escape sequence cannot reach your terminal and drive it — OSC 52 in a note body would otherwise rewrite your system clipboard.

## Diagnostics

`doctor --json` and `paths --json` print absolute paths, every registered project root, branch names, and Git remote fingerprints. Credentials are stripped from remotes (`https://user:token@host/repo` is recorded as `host/repo`), but the rest is local detail worth reviewing before pasting the output into an issue.

No support window is promised before the first stable release. Security fixes will be documented in the changelog without exposing exploit details prematurely.
