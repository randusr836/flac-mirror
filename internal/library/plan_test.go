package library

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuild(t *testing.T) {
	src := t.TempDir()
	dest := t.TempDir()

	// A track that needs transcoding (no destination yet).
	mustWriteFile(t, filepath.Join(src, "Artist", "Album", "01 New.flac"), "flac")

	// A track that's already up to date: write the destination mp3 first,
	// then the source with an older mtime.
	upToDateSrc := filepath.Join(src, "Artist", "Album", "02 Current.flac")
	upToDateDest := filepath.Join(dest, "Artist", "Album", "02 Current.mp3")
	mustWriteFile(t, upToDateDest, "mp3")
	mustWriteFile(t, upToDateSrc, "flac")
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(upToDateSrc, old, old); err != nil {
		t.Fatal(err)
	}

	// A cover art file with no destination yet, so it needs copying.
	mustWriteFile(t, filepath.Join(src, "Artist", "Album", "cover.jpg"), "jpg")

	// An orphan on the destination with no corresponding source file.
	orphan := filepath.Join(dest, "Artist", "Album", "old track.mp3")
	mustWriteFile(t, orphan, "stale")

	plan, err := Build(src, dest)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	if len(plan.ToTranscode) != 1 || plan.ToTranscode[0].RelPath != filepath.Join("Artist", "Album", "01 New.mp3") {
		t.Errorf("ToTranscode = %+v, want just 01 New.mp3", plan.ToTranscode)
	}
	if len(plan.UpToDate) != 1 || plan.UpToDate[0].RelPath != filepath.Join("Artist", "Album", "02 Current.mp3") {
		t.Errorf("UpToDate = %+v, want just 02 Current.mp3", plan.UpToDate)
	}
	if len(plan.ToCopy) != 1 || plan.ToCopy[0].RelPath != filepath.Join("Artist", "Album", "cover.jpg") {
		t.Errorf("ToCopy = %+v, want just cover.jpg", plan.ToCopy)
	}
	if len(plan.ToDelete) != 1 || plan.ToDelete[0] != orphan {
		t.Errorf("ToDelete = %+v, want just %q", plan.ToDelete, orphan)
	}
}

func TestBuildDestNotYetCreated(t *testing.T) {
	src := t.TempDir()
	dest := filepath.Join(t.TempDir(), "does-not-exist-yet")
	mustWriteFile(t, filepath.Join(src, "Artist", "Album", "01 Track.flac"), "flac")

	plan, err := Build(src, dest)
	if err != nil {
		t.Fatalf("Build with missing dest root: %v", err)
	}
	if len(plan.ToTranscode) != 1 {
		t.Errorf("ToTranscode = %+v, want 1 item", plan.ToTranscode)
	}
	if len(plan.ToDelete) != 0 {
		t.Errorf("ToDelete = %+v, want none", plan.ToDelete)
	}
}

func TestFindOrphansSkipsReservedDirs(t *testing.T) {
	dest := t.TempDir()
	mustWriteFile(t, filepath.Join(dest, ".Spotlight-V100", "store.db"), "index")
	mustWriteFile(t, filepath.Join(dest, "Artist", "Album", "track.mp3"), "mp3")

	orphans, err := findOrphans(dest, map[string]bool{
		filepath.Join(dest, "Artist", "Album", "track.mp3"): true,
	})
	if err != nil {
		t.Fatalf("findOrphans: %v", err)
	}
	if len(orphans) != 0 {
		t.Errorf("findOrphans returned %v, want none (Spotlight dir should be skipped, track.mp3 is expected)", orphans)
	}
}

func TestPruneEmptyDirs(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "KeepMe", "track.mp3"), "mp3")
	if err := os.MkdirAll(filepath.Join(root, "EmptyArtist", "EmptyAlbum"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := PruneEmptyDirs(root); err != nil {
		t.Fatalf("PruneEmptyDirs: %v", err)
	}

	if _, err := os.Stat(filepath.Join(root, "EmptyArtist")); !os.IsNotExist(err) {
		t.Errorf("expected EmptyArtist to be pruned, stat err = %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "KeepMe", "track.mp3")); err != nil {
		t.Errorf("expected KeepMe/track.mp3 to survive, stat err = %v", err)
	}
}
