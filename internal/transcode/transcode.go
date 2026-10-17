// Package transcode shells out to ffmpeg to convert FLAC files to MP3.
package transcode

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ToMP3 transcodes srcFlac to destMp3 at the given CBR bitrate (e.g.
// "320k"), creating destMp3's parent directory if needed. Source tags are
// preserved via ffmpeg's metadata copy.
func ToMP3(ctx context.Context, srcFlac, destMp3, bitrate string) error {
	if err := os.MkdirAll(filepath.Dir(destMp3), 0o755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-i", srcFlac,
		"-map_metadata", "0",
		"-vn",
		"-codec:a", "libmp3lame",
		"-b:a", bitrate,
		"-id3v2_version", "3",
		destMp3,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Don't leave a partial/empty file behind: it would otherwise be
		// treated as up to date and never retried on the next run.
		os.Remove(destMp3)
		return fmt.Errorf("ffmpeg failed on %s: %w (%s)", srcFlac, err, summarize(stderr.String()))
	}
	return nil
}

// summarize collapses ffmpeg's (often many-line) stderr output down to its
// last few non-empty lines, joined onto one line. ffmpeg's own error is
// usually near the end, and a multi-line message breaks the fixed-height
// redraw of the progress TUI, so callers should never surface the raw
// stderr blob directly.
func summarize(stderr string) string {
	var lines, interesting []string
	for _, line := range strings.Split(stderr, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lines = append(lines, line)
		lower := strings.ToLower(line)
		if strings.Contains(lower, "error") || strings.Contains(lower, "invalid") {
			interesting = append(interesting, line)
		}
	}

	// ffmpeg's most actionable line is usually one mentioning the actual
	// error/invalid-data reason; fall back to the tail of the output
	// (typically "Conversion failed!"-type lines) if nothing matched.
	if len(interesting) > 0 {
		lines = interesting
	}
	if len(lines) == 0 {
		return "no output"
	}

	const keep = 2
	if len(lines) > keep {
		lines = lines[len(lines)-keep:]
	}
	return strings.Join(lines, " | ")
}
