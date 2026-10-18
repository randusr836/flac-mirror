package library

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsFlac(t *testing.T) {
	cases := map[string]bool{
		"Track.flac":   true,
		"Track.FLAC":   true,
		"Track.mp3":    false,
		"Track":        false,
		"._Track.flac": true, // extension check alone; AppleDouble filtering happens separately
	}
	for name, want := range cases {
		if got := isFlac(name); got != want {
			t.Errorf("isFlac(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestIsPlaylist(t *testing.T) {
	cases := map[string]bool{
		"Album.m3u":  true,
		"Album.M3U8": true,
		"Album.txt":  false,
	}
	for name, want := range cases {
		if got := isPlaylist(name); got != want {
			t.Errorf("isPlaylist(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestIsCoverArt(t *testing.T) {
	cases := map[string]bool{
		"cover.jpg":    true,
		"Cover.JPEG":   true,
		"folder.png":   true,
		"front.jpg":    true,
		"albumart.jpg": true,
		"random.jpg":   false, // wrong basename
		"cover.gif":    false, // wrong extension
		"cover.txt":    false,
	}
	for name, want := range cases {
		if got := isCoverArt(name); got != want {
			t.Errorf("isCoverArt(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestIsAppleDouble(t *testing.T) {
	cases := map[string]bool{
		"._Track.flac": true,
		"._cover.jpg":  true,
		"Track.flac":   false,
		".hidden":      false,
	}
	for name, want := range cases {
		if got := isAppleDouble(name); got != want {
			t.Errorf("isAppleDouble(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestScan(t *testing.T) {
	src := t.TempDir()
	dest := "/dest" // Scan never touches dest, only computes paths under it

	mustWriteFile(t, filepath.Join(src, "Artist", "Album", "01 Track.flac"), "flac data")
	mustWriteFile(t, filepath.Join(src, "Artist", "Album", "Album.m3u"), "01 Track.flac\n")
	mustWriteFile(t, filepath.Join(src, "Artist", "Album", "cover.jpg"), "jpg data")
	mustWriteFile(t, filepath.Join(src, "Artist", "Album", "._01 Track.flac"), "appledouble junk")
	mustWriteFile(t, filepath.Join(src, "Artist", "Album", "notes.txt"), "ignored")

	items, err := Scan(src, dest)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}

	byRel := map[string]Item{}
	for _, it := range items {
		byRel[it.RelPath] = it
	}

	if len(items) != 3 {
		t.Fatalf("got %d items, want 3 (flac+m3u+cover, appledouble/txt excluded): %+v", len(items), items)
	}

	track, ok := byRel[filepath.Join("Artist", "Album", "01 Track.mp3")]
	if !ok {
		t.Fatal("missing transcode item for 01 Track.mp3")
	}
	if track.Kind != KindTranscode {
		t.Errorf("track.Kind = %v, want KindTranscode", track.Kind)
	}
	if track.DestPath != filepath.Join(dest, "Artist", "Album", "01 Track.mp3") {
		t.Errorf("track.DestPath = %q", track.DestPath)
	}

	playlist, ok := byRel[filepath.Join("Artist", "Album", "Album.m3u")]
	if !ok || playlist.Kind != KindPlaylist {
		t.Errorf("missing/wrong-kind playlist item: %+v ok=%v", playlist, ok)
	}

	cover, ok := byRel[filepath.Join("Artist", "Album", "cover.jpg")]
	if !ok || cover.Kind != KindCoverArt {
		t.Errorf("missing/wrong-kind cover item: %+v ok=%v", cover, ok)
	}
}

func TestNeedsWork(t *testing.T) {
	dest := t.TempDir()
	destFile := filepath.Join(dest, "track.mp3")

	t.Run("missing destination", func(t *testing.T) {
		item := Item{DestPath: destFile, SourceModTime: time.Now()}
		stale, err := NeedsWork(item)
		if err != nil {
			t.Fatalf("NeedsWork: %v", err)
		}
		if !stale {
			t.Error("expected stale=true for missing destination")
		}
	})

	mustWriteFile(t, destFile, "existing mp3")
	destInfo, err := os.Stat(destFile)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("source older than destination", func(t *testing.T) {
		item := Item{DestPath: destFile, SourceModTime: destInfo.ModTime().Add(-time.Hour)}
		stale, err := NeedsWork(item)
		if err != nil {
			t.Fatalf("NeedsWork: %v", err)
		}
		if stale {
			t.Error("expected stale=false when source predates destination")
		}
	})

	t.Run("source newer than destination", func(t *testing.T) {
		item := Item{DestPath: destFile, SourceModTime: destInfo.ModTime().Add(time.Hour)}
		stale, err := NeedsWork(item)
		if err != nil {
			t.Fatalf("NeedsWork: %v", err)
		}
		if !stale {
			t.Error("expected stale=true when source is newer than destination")
		}
	})
}

func mustWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
