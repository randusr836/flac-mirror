package library

import (
	"fmt"
	"io/fs"
	"os"
)

// Plan is the full set of work needed to bring destRoot in line with
// sourceRoot: which items need (re)generating, which are already current,
// and which files on the destination are orphaned and should be deleted.
type Plan struct {
	ToTranscode []Item
	ToCopy      []Item // playlists + cover art
	UpToDate    []Item
	ToDelete    []string // absolute paths under destRoot with no matching source item
}

// Build scans sourceRoot, compares it against the current state of
// destRoot, and returns the resulting Plan.
func Build(sourceRoot, destRoot string) (Plan, error) {
	items, err := Scan(sourceRoot, destRoot)
	if err != nil {
		return Plan{}, err
	}

	var plan Plan
	expected := make(map[string]bool, len(items))

	for _, item := range items {
		expected[item.DestPath] = true

		stale, err := NeedsWork(item)
		if err != nil {
			return Plan{}, err
		}
		if !stale {
			plan.UpToDate = append(plan.UpToDate, item)
			continue
		}
		if item.Kind == KindTranscode {
			plan.ToTranscode = append(plan.ToTranscode, item)
		} else {
			plan.ToCopy = append(plan.ToCopy, item)
		}
	}

	toDelete, err := findOrphans(destRoot, expected)
	if err != nil {
		return Plan{}, err
	}
	plan.ToDelete = toDelete

	return plan, nil
}

// findOrphans walks destRoot and returns every file not present in
// expected. destRoot may not exist yet, which is not an error.
func findOrphans(destRoot string, expected map[string]bool) ([]string, error) {
	var orphans []string

	err := walkSkippingReserved(destRoot, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() {
			return nil
		}
		if !expected[path] {
			orphans = append(orphans, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scanning %s for orphaned files: %w", destRoot, err)
	}

	return orphans, nil
}

// PruneEmptyDirs removes any directory under root that is empty (or only
// contains other empty directories), deepest first. Call it after deleting
// orphaned files so album/artist folders left with nothing in them don't
// linger.
func PruneEmptyDirs(root string) error {
	var dirs []string
	err := walkSkippingReserved(root, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() && path != root {
			dirs = append(dirs, path)
		}
		return nil
	})
	if err != nil {
		return err
	}

	// Deepest paths first, so parents become empty only after their
	// children have already been removed.
	for i := len(dirs) - 1; i >= 0; i-- {
		entries, err := os.ReadDir(dirs[i])
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return err
		}
		if len(entries) == 0 {
			if err := os.Remove(dirs[i]); err != nil {
				return err
			}
		}
	}
	return nil
}
