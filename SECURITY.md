# Security Policy

## Reporting a vulnerability

Please don't open a public GitHub issue for security vulnerabilities.

Instead, use GitHub's private reporting:
[Report a vulnerability](https://github.com/randusr836/flac-mirror/security/advisories/new)
(Security tab → Advisories → "Report a vulnerability").

Please include:

- A description of the issue and its potential impact
- Steps to reproduce, or a minimal example
- The flac-mirror version/commit and OS you tested on

## Scope

flac-mirror is a local CLI tool that shells out to `ffmpeg` and reads/writes
files on paths you give it. There's no network service and no handling of
untrusted remote input — the main things worth reporting are anything that
could lead to unintended file access/deletion outside the given source/dest
directories, or command-injection-style issues in how arguments are passed
to `ffmpeg`.

## Response

This is a solo-maintained open source project — there's no guaranteed SLA,
but reports will be acknowledged and addressed as soon as reasonably
possible.
