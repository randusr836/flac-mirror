// Package copyfile handles the non-transcoding destination writes: plain
// file copies (cover art) and playlist rewriting (.m3u/.m3u8).
package copyfile

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Plain copies src to dest byte-for-byte, creating dest's parent directory
// if needed.
func Plain(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dest, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dest)
		return fmt.Errorf("copying %s to %s: %w", src, dest, err)
	}
	return out.Close()
}

// Playlist copies src (a .m3u/.m3u8 file) to dest, rewriting any .flac
// file references on non-comment lines to .mp3 so the playlist plays the
// transcoded files on the destination.
func Playlist(src, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("creating destination directory: %w", err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("opening %s: %w", src, err)
	}
	defer in.Close()

	out, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dest, err)
	}
	defer out.Close()

	scanner := bufio.NewScanner(in)
	writer := bufio.NewWriter(out)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			fmt.Fprintln(writer, line)
			continue
		}

		if ext := filepath.Ext(line); strings.EqualFold(ext, ".flac") {
			line = line[:len(line)-len(ext)] + ".mp3"
		}
		fmt.Fprintln(writer, line)
	}
	if err := scanner.Err(); err != nil {
		out.Close()
		os.Remove(dest)
		return fmt.Errorf("reading %s: %w", src, err)
	}

	if err := writer.Flush(); err != nil {
		out.Close()
		os.Remove(dest)
		return fmt.Errorf("writing %s: %w", dest, err)
	}
	return out.Close()
}
