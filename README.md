# flac-mirror

[![tests](https://github.com/randusr836/flac-mirror/actions/workflows/test.yml/badge.svg)](https://github.com/randusr836/flac-mirror/actions/workflows/test.yml)

A CLI tool that mirrors a FLAC music library onto a destination (e.g. an SD
card) as 320kbps MP3s. It transcodes in parallel across CPU cores and
shows live progress with an ETA in a terminal UI.

The destination is kept as an exact mirror of the source: anything removed
or renamed on the source is deleted from the destination too.

## Demo
https://github.com/user-attachments/assets/11cb5058-59fc-4cee-9e90-50a791796a82

## Features

- Transcodes `.flac` → `.mp3` (320kbps CBR) via `ffmpeg`, preserving tags
- Parallel transcoding across all CPU cores (configurable)
- Mirrors a `Artist/Album/Track` folder structure
- Rewrites `.m3u`/`.m3u8` playlists so they point at the new `.mp3` files
- Copies cover art (`cover`/`folder`/`front`/`albumart`, `.jpg`/`.jpeg`/`.png`)
- Skips files that are already up to date — only changed/new files are
  reprocessed on repeat runs
- Full mirror: deletes anything on the destination with no corresponding
  source file, and prunes any directories left empty
- Live TUI with a progress bar, ETA, and in-flight file list (falls back
  to plain log lines if output isn't a terminal, e.g. when redirected)

## Requirements

- [Go](https://go.dev/) 1.24+ (only to build it from source — not needed if you download a release binary)
- [ffmpeg](https://ffmpeg.org/) on your `PATH`

### Installing ffmpeg

**macOS** (Homebrew):
```sh
brew install ffmpeg
```

**Windows** (winget or Chocolatey):
```sh
winget install ffmpeg
# or
choco install ffmpeg
```
Alternatively, download a static build from
[ffmpeg.org/download.html](https://ffmpeg.org/download.html) and add its
`bin\` folder to your `PATH`.

**Linux:**
```sh
sudo apt install ffmpeg        # Debian/Ubuntu
sudo dnf install ffmpeg        # Fedora
sudo pacman -S ffmpeg          # Arch
```

If `ffmpeg` isn't found, flac-mirror prints the relevant instructions for
your OS and exits.

## Installing

Download the archive for your platform from the
[releases page](https://github.com/randusr836/flac-mirror/releases),
extract it, and run the `flacmirror` binary (`flacmirror.exe` on Windows)
directly — it has no runtime dependencies besides `ffmpeg`.

### Building from source

```sh
go build -o flacmirror ./cmd/flacmirror
```

## Usage

```sh
./flacmirror [flags] <source-dir> <dest-dir>
```

**Example:**
```sh
./flacmirror /Volumes/MusicHDD/FLAC /Volumes/SDCARD/Music
```

On first run this transcodes everything. On later runs, only new or
changed files are processed, and anything removed from the source is
deleted from the destination.

### Flags

| Flag        | Default        | Description                                        |
|-------------|----------------|-----------------------------------------------------|
| `-j`        | number of CPUs | Number of parallel transcode workers                |
| `-bitrate`  | `320k`         | MP3 CBR bitrate passed to ffmpeg                    |
| `-dry-run`  | `false`        | Show what would change without touching any files   |
| `-y`        | `false`        | Skip the confirmation prompt before deleting files  |

### Expected library layout

flac-mirror expects a layout like:

```
FLAC/
├── Artist One/
│   └── Album A/
│       ├── 01 Track One.flac
│       ├── 02 Track Two.flac
│       ├── Album A.m3u
│       └── cover.jpg
└── Artist Two/
    └── Album B/
        └── 01 Solo.flac
```

It recurses through any depth of subfolders, so this layout isn't strictly
required — it just finds every `.flac`, `.m3u`/`.m3u8`, and cover-art file
and mirrors each one to the same relative path on the destination (with
`.flac` replaced by `.mp3`).

### Deletions and dry runs

Because the destination is kept as a full mirror, flac-mirror will delete
files on the destination that no longer have a matching source file. Before
doing so (unless `-y` is passed) it asks for confirmation:

```
This will permanently delete 12 file(s) from /Volumes/SDCARD/Music. Continue? [y/N]
```

To see what a run would do without changing anything:

```sh
./flacmirror -dry-run /Volumes/MusicHDD/FLAC /Volumes/SDCARD/Music
```

## Acknowledgments

Built with the help of [Claude Code](https://claude.com/claude-code).

## Project layout

```
cmd/flacmirror/       CLI entry point (flags, confirmation prompt, wiring)
internal/ffmpegutil/  ffmpeg detection + install instructions
internal/library/     scans the source tree and builds the sync plan
internal/transcode/   runs ffmpeg to produce MP3s
internal/copyfile/    plain file copies + playlist rewriting
internal/engine/      worker pool that executes a plan, emits progress events
internal/tui/         Bubble Tea progress UI
```
