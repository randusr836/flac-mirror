package library

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCheckWritable(t *testing.T) {
	dest := t.TempDir()
	if err := CheckWritable(dest); err != nil {
		t.Fatalf("CheckWritable(%q) = %v, want nil", dest, err)
	}
	entries, err := os.ReadDir(dest)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("expected probe file to be cleaned up, found %d entries", len(entries))
	}
}

func TestCheckWritableMissingDest(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "not", "yet", "created")
	if err := CheckWritable(dest); err != nil {
		t.Fatalf("CheckWritable(%q) = %v, want nil", dest, err)
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("CheckWritable should not create dest, stat err = %v", err)
	}
}

func TestCheckWritableDestIsFile(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(dest, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckWritable(dest); err == nil {
		t.Fatal("expected an error when dest is a file")
	}
}

func TestCheckWritableNoPermission(t *testing.T) {
	if runtime.GOOS == "windows" || os.Geteuid() == 0 {
		t.Skip("directory permissions are not enforced here")
	}
	dest := t.TempDir()
	if err := os.Chmod(dest, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dest, 0o755) })

	err := CheckWritable(dest)
	if err == nil || !strings.Contains(err.Error(), "no permission") {
		t.Fatalf("CheckWritable = %v, want a permission error", err)
	}
}
