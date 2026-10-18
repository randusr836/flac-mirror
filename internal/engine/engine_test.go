package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"flacmirror/internal/library"
)

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// collect drains events into a slice, for assertions after Run returns.
func collect(events <-chan Event) []Event {
	var got []Event
	for ev := range events {
		got = append(got, ev)
	}
	return got
}

func TestRunCopyAndDelete(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()

	coverSrc := filepath.Join(src, "cover.jpg")
	mustWriteFile(t, coverSrc, "jpg bytes")
	coverDest := filepath.Join(dest, "cover.jpg")

	playlistSrc := filepath.Join(src, "Album.m3u")
	mustWriteFile(t, playlistSrc, "01 Track.flac\n")
	playlistDest := filepath.Join(dest, "Album.m3u")

	orphan := filepath.Join(dest, "Artist", "stale.mp3")
	mustWriteFile(t, orphan, "stale")

	plan := library.Plan{
		ToCopy: []library.Item{
			{RelPath: "cover.jpg", SourcePath: coverSrc, DestPath: coverDest, Kind: library.KindCoverArt},
			{RelPath: "Album.m3u", SourcePath: playlistSrc, DestPath: playlistDest, Kind: library.KindPlaylist},
		},
		ToDelete: []string{orphan},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	events := make(chan Event)
	go Run(ctx, plan, Options{Workers: 2, DestRoot: dest}, events)
	got := collect(events)

	// Destination files should now exist with the right content.
	if b, err := os.ReadFile(coverDest); err != nil || string(b) != "jpg bytes" {
		t.Errorf("cover art not copied correctly: err=%v content=%q", err, b)
	}
	if b, err := os.ReadFile(playlistDest); err != nil || string(b) != "01 Track.mp3\n" {
		t.Errorf("playlist not rewritten correctly: err=%v content=%q", err, b)
	}

	// The orphaned file (and its now-empty parent dir) should be gone.
	if _, err := os.Stat(orphan); !os.IsNotExist(err) {
		t.Errorf("expected orphan to be deleted, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, "Artist")); !os.IsNotExist(err) {
		t.Errorf("expected emptied Artist dir to be pruned, stat err = %v", err)
	}

	assertEventSequence(t, got)
}

func assertEventSequence(t *testing.T, got []Event) {
	t.Helper()
	if len(got) == 0 || got[0].Type != EventPlanReady {
		t.Fatalf("expected first event to be EventPlanReady, got %+v", firstOrNil(got))
	}
	if got[len(got)-1].Type != EventAllDone {
		t.Fatalf("expected last event to be EventAllDone, got %+v", got[len(got)-1])
	}

	var starts, dones, deletes int
	for _, ev := range got {
		switch ev.Type {
		case EventJobStart:
			starts++
		case EventJobDone:
			dones++
			if ev.Err != nil {
				t.Errorf("unexpected job error for %s: %v", ev.Item.RelPath, ev.Err)
			}
		case EventDeleteDone:
			deletes++
		}
	}
	if starts != 2 || dones != 2 {
		t.Errorf("starts=%d dones=%d, want 2 and 2", starts, dones)
	}
	if deletes != 1 {
		t.Errorf("deletes=%d, want 1", deletes)
	}
}

func firstOrNil(events []Event) any {
	if len(events) == 0 {
		return nil
	}
	return events[0]
}

func TestRunReportsJobErrors(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()

	// A cover-art "source" that doesn't exist: the copy should fail and
	// surface as an EventJobDone with a non-nil Err, without Run getting
	// stuck.
	missingSrc := filepath.Join(src, "missing.jpg")
	plan := library.Plan{
		ToCopy: []library.Item{
			{RelPath: "missing.jpg", SourcePath: missingSrc, DestPath: filepath.Join(dest, "missing.jpg"), Kind: library.KindCoverArt},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	events := make(chan Event)
	go Run(ctx, plan, Options{Workers: 1, DestRoot: dest}, events)
	got := collect(events)

	var sawErr bool
	for _, ev := range got {
		if ev.Type == EventJobDone && ev.Err != nil {
			sawErr = true
		}
	}
	if !sawErr {
		t.Error("expected an EventJobDone with a non-nil Err for the missing source file")
	}
}
