package library

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// skippedDirNames are OS/filesystem-managed directories that routinely
// turn up at the root of external and network volumes (Spotlight indexes,
// the macOS/Windows trash, filesystem event logs, Windows system
// metadata, ...). They aren't part of any music library, and on removable
// media they're frequently not even readable by the current user, so
// walks skip them outright rather than erroring out.
var skippedDirNames = map[string]bool{
	".spotlight-v100":           true,
	".trashes":                  true,
	".fseventsd":                true,
	".temporaryitems":           true,
	".documentrevisions-v100":   true,
	"system volume information": true,
	"$recycle.bin":              true,
}

func isSkippedDir(name string) bool {
	return skippedDirNames[strings.ToLower(name)]
}

// walkSkippingReserved runs filepath.WalkDir over root, skipping OS-managed
// directories by name and tolerating permission errors and a missing root
// (either is treated as "nothing here" rather than a fatal error) so a
// scan of an external/removable volume doesn't abort over metadata the
// current user isn't allowed to read.
func walkSkippingReserved(root string, fn fs.WalkDirFunc) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) || os.IsPermission(err) {
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			return err
		}
		if d.IsDir() && path != root && isSkippedDir(d.Name()) {
			return filepath.SkipDir
		}
		return fn(path, d, err)
	})
}
