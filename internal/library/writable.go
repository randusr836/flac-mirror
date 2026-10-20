package library

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

// CheckWritable verifies that files can be created under dest, so that a
// read-only or permission-locked destination is reported once up front
// rather than as a failure for every file in the plan. If dest does not exist
// yet, its nearest existing ancestor is checked instead, since that is where
// the mirror's directories would be created.
func CheckWritable(dest string) error {
	dir, err := nearestExistingDir(dest)
	if err != nil {
		return fmt.Errorf("checking destination %q: %w", dest, err)
	}

	f, err := os.CreateTemp(dir, ".flacmirror-write-test-*")
	if err != nil {
		return describeWriteErr(dest, err)
	}
	name := f.Name()
	f.Close()
	_ = os.Remove(name)
	return nil
}

func nearestExistingDir(path string) (string, error) {
	path = filepath.Clean(path)
	for {
		info, err := os.Stat(path)
		if err == nil {
			if !info.IsDir() {
				return "", fmt.Errorf("%q is not a directory", path)
			}
			return path, nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(path)
		if parent == path {
			return "", err
		}
		path = parent
	}
}

func describeWriteErr(dest string, err error) error {
	switch {
	case errors.Is(err, syscall.EROFS):
		msg := fmt.Sprintf("destination %q is on a read-only filesystem.\n", dest) +
			"  - If it is an SD card, check that its lock switch is not set to \"Lock\".\n"
		if runtime.GOOS == "darwin" {
			msg += "  - macOS cannot write to NTFS-formatted drives; reformat it as exFAT or FAT32.\n"
		}
		msg += "  - Otherwise, try ejecting and reinserting the drive, or remount it read-write"
		return errors.New(msg)
	case errors.Is(err, fs.ErrPermission):
		return fmt.Errorf("no permission to write to destination %q.\n"+
			"  Check the ownership and permissions of the destination directory", dest)
	default:
		return fmt.Errorf("cannot write to destination %q: %w", dest, err)
	}
}
