# Contributing

Thanks for considering a contribution to flac-mirror!

## Getting set up

You'll need:

- [Go](https://go.dev/) 1.24+
- [ffmpeg](https://ffmpeg.org/) on your `PATH` (see the README for
  per-platform install instructions)

```sh
git clone https://github.com/randusr836/flac-mirror.git
cd flac-mirror
go build -o flacmirror ./cmd/flacmirror
```

## Making changes

- Run `gofmt -l .` and fix anything it lists — CI rejects unformatted code.
- Run `go vet ./...` — CI runs this too.
- Run `go test ./...` before opening a PR. If `ffmpeg` is installed, this
  also exercises the real transcode path; if it isn't, those specific
  tests are skipped rather than failing.
- Keep changes focused. If you're fixing a bug, a test that reproduces it
  first is appreciated.

## Submitting a pull request

1. Fork the repo and create a branch off `main`.
2. Make your change, with tests where it makes sense.
3. Make sure `gofmt`, `go vet`, and `go test ./...` are all clean.
4. Open a PR describing what changed and why. Link any related issue.

CI (gofmt/vet/test) runs automatically on every PR.

## Reporting bugs / requesting features

Use the issue templates — they ask for the information needed to act on
a report (OS, Go version, exact command, etc.).

## Security issues

Please don't open a public issue for a security vulnerability — see
[SECURITY.md](SECURITY.md).
