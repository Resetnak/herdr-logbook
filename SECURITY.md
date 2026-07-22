# Security

Herdr Memory is offline at runtime and stores canonical user data as Markdown. Please report a suspected vulnerability privately through GitHub Security Advisories rather than a public issue. Include the affected version, platform, reproduction steps, and whether a crafted repository, note, remote URL, or environment variable is required.

Install scripts are the only networked component. They download a version-pinned release asset from this repository and verify its SHA-256 checksum before extraction. The application does not execute note content, interpolate titles through a shell, emit selected terminal content in diagnostics, or persist credentials from Git remotes.

No support window is promised before the first stable release. Security fixes will be documented in the changelog without exposing exploit details prematurely.
