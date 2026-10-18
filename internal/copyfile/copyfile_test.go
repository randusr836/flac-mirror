package copyfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPlain(t *testing.T) {
	src := filepath.Join(t.TempDir(), "cover.jpg")
	if err := os.WriteFile(src, []byte("fake jpg bytes"), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "nested", "deep", "cover.jpg")
	if err := Plain(src, dest); err != nil {
		t.Fatalf("Plain: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading copied file: %v", err)
	}
	if string(got) != "fake jpg bytes" {
		t.Errorf("copied content = %q, want %q", got, "fake jpg bytes")
	}
}

func TestPlainMissingSource(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "cover.jpg")
	if err := Plain(filepath.Join(t.TempDir(), "does-not-exist.jpg"), dest); err == nil {
		t.Fatal("expected error for missing source, got nil")
	}
}

func TestPlaylist(t *testing.T) {
	src := filepath.Join(t.TempDir(), "Album.m3u")
	content := "#EXTM3U\n" +
		"#EXTINF:123,Artist - Track One\n" +
		"01 Track One.flac\n" +
		"\n" +
		"02 Track Two.FLAC\n" + // uppercase extension should still be rewritten
		"not-a-flac.txt\n"
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	dest := filepath.Join(t.TempDir(), "Album.m3u")
	if err := Playlist(src, dest); err != nil {
		t.Fatalf("Playlist: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}

	want := "#EXTM3U\n" +
		"#EXTINF:123,Artist - Track One\n" +
		"01 Track One.mp3\n" +
		"\n" +
		"02 Track Two.mp3\n" +
		"not-a-flac.txt\n"
	if string(got) != want {
		t.Errorf("rewritten playlist = %q, want %q", got, want)
	}
}

func TestPlaylistMissingSource(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "Album.m3u")
	if err := Playlist(filepath.Join(t.TempDir(), "does-not-exist.m3u"), dest); err == nil {
		t.Fatal("expected error for missing source, got nil")
	}
}
