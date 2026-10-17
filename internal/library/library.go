// Package library scans a source music tree (artist/album/track layout)
// and figures out what needs to exist on the mirrored destination.
package library

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Kind identifies what kind of destination artifact a source file maps to.
type Kind int

const (
	// KindTranscode is a .flac source that becomes a .mp3 on the destination.
	KindTranscode Kind = iota
	// KindPlaylist is a .m3u/.m3u8 playlist, copied with .flac references
	// inside it rewritten to .mp3.
	KindPlaylist
	// KindCoverArt is a cover-image file, copied byte-for-byte.
	KindCoverArt
)

func (k Kind) String() string {
	switch k {
	case KindTranscode:
		return "transcode"
	case KindPlaylist:
		return "playlist"
	case KindCoverArt:
		return "cover art"
	default:
		return "unknown"
	}
}

// Item is one file discovered on the source side, along with where it
// should end up on the destination.
type Item struct {
	RelPath       string // destination-relative path, e.g. "Artist/Album/01 Track.mp3"
	SourcePath    string
	DestPath      string
	Kind          Kind
	SourceModTime time.Time
}

var coverArtBaseNames = map[string]bool{
	"cover":    true,
	"folder":   true,
	"front":    true,
	"albumart": true,
}

var coverArtExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
}

func isCoverArt(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	if !coverArtExts[ext] {
		return false
	}
	base := strings.ToLower(strings.TrimSuffix(name, filepath.Ext(name)))
	return coverArtBaseNames[base]
}

func isPlaylist(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".m3u" || ext == ".m3u8"
}

func isFlac(name string) bool {
	return strings.ToLower(filepath.Ext(name)) == ".flac"
}

// isAppleDouble reports whether name is a macOS AppleDouble sidecar file
// (e.g. "._Track.flac") — metadata macOS writes alongside real files on
// non-native filesystems (exFAT/FAT32 SD cards, network shares). These
// share their real counterpart's extension but aren't valid audio/text,
// so they must never be scanned as if they were.
func isAppleDouble(name string) bool {
	return strings.HasPrefix(name, "._")
}

// Scan walks sourceRoot and returns every FLAC, playlist and cover-art file
// found, mapped to its destination-relative path and kind.
func Scan(sourceRoot, destRoot string) ([]Item, error) {
	var items []Item

	err := walkSkippingReserved(sourceRoot, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}

		name := d.Name()
		if isAppleDouble(name) {
			return nil
		}

		var kind Kind
		var destRel string

		switch {
		case isFlac(name):
			kind = KindTranscode
		case isPlaylist(name):
			kind = KindPlaylist
		case isCoverArt(name):
			kind = KindCoverArt
		default:
			return nil
		}

		relPath, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return fmt.Errorf("computing relative path for %s: %w", path, err)
		}

		if kind == KindTranscode {
			destRel = relPath[:len(relPath)-len(filepath.Ext(relPath))] + ".mp3"
		} else {
			destRel = relPath
		}

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", path, err)
		}

		items = append(items, Item{
			RelPath:       destRel,
			SourcePath:    path,
			DestPath:      filepath.Join(destRoot, destRel),
			Kind:          kind,
			SourceModTime: info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning %s: %w", sourceRoot, err)
	}

	return items, nil
}

// NeedsWork reports whether an item's destination is missing or older than
// its source, i.e. whether it needs to be (re)generated.
func NeedsWork(item Item) (bool, error) {
	info, err := os.Stat(item.DestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return true, nil
		}
		return false, fmt.Errorf("stat %s: %w", item.DestPath, err)
	}
	return item.SourceModTime.After(info.ModTime()), nil
}
