package transcode

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSummarize(t *testing.T) {
	cases := []struct {
		name   string
		stderr string
		want   string
	}{
		{
			name:   "empty output",
			stderr: "",
			want:   "no output",
		},
		{
			name:   "blank lines only",
			stderr: "\n\n   \n",
			want:   "no output",
		},
		{
			name: "prefers lines mentioning error or invalid",
			stderr: "ffmpeg version 9.0.2\n" +
				"  built with clang\n" +
				"[in#0/flac] Could not find codec parameters: unspecified sample format\n" +
				"Stream mapping:\n" +
				"[aost#0:0] Task finished with error code: -22 (Invalid argument)\n" +
				"[aost#0:0] Terminating thread with return code -22 (Invalid argument)\n" +
				"size=0KiB time=N/A\n" +
				"Conversion failed!\n",
			want: "[aost#0:0] Task finished with error code: -22 (Invalid argument) | [aost#0:0] Terminating thread with return code -22 (Invalid argument)",
		},
		{
			name: "falls back to last lines when nothing matches",
			stderr: "line one\n" +
				"line two\n" +
				"line three\n",
			want: "line two | line three",
		},
		{
			name:   "single line",
			stderr: "   just one line   \n",
			want:   "just one line",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := summarize(tc.stderr); got != tc.want {
				t.Errorf("summarize(%q) = %q, want %q", tc.stderr, got, tc.want)
			}
		})
	}
}

func TestToMP3(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH, skipping integration test")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "track.flac")

	gen := exec.Command("ffmpeg", "-y", "-loglevel", "error",
		"-f", "lavfi", "-i", "sine=frequency=440:duration=1", src)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("generating test flac: %v\n%s", err, out)
	}

	dest := filepath.Join(dir, "out", "track.mp3")
	if err := ToMP3(context.Background(), src, dest, "320k"); err != nil {
		t.Fatalf("ToMP3: %v", err)
	}

	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("stat output: %v", err)
	}
	if info.Size() == 0 {
		t.Error("output mp3 is empty")
	}
}

func TestToMP3FailureCleansUpOutput(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not on PATH, skipping integration test")
	}

	dir := t.TempDir()
	src := filepath.Join(dir, "corrupt.flac")
	if err := os.WriteFile(src, []byte("not actually flac data"), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(dir, "corrupt.mp3")
	err := ToMP3(context.Background(), src, dest, "320k")
	if err == nil {
		t.Fatal("expected ToMP3 to fail on corrupt input")
	}
	if !strings.Contains(err.Error(), "ffmpeg failed") {
		t.Errorf("error = %v, want it to mention ffmpeg failure", err)
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Errorf("expected no leftover output file, stat err = %v", statErr)
	}
}
