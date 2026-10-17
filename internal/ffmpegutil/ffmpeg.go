// Package ffmpegutil checks for and reports on the ffmpeg binary this tool
// shells out to for all FLAC->MP3 transcoding.
package ffmpegutil

import (
	"fmt"
	"os/exec"
	"runtime"
)

// ErrNotFound is returned by Check when ffmpeg isn't on PATH.
type ErrNotFound struct{}

func (ErrNotFound) Error() string { return "ffmpeg not found on PATH" }

// Check verifies that an ffmpeg binary is available on PATH.
func Check() error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return ErrNotFound{}
	}
	return nil
}

// InstallInstructions returns OS-specific instructions for installing
// ffmpeg, for display when Check fails.
func InstallInstructions() string {
	switch runtime.GOOS {
	case "darwin":
		return "ffmpeg was not found on your PATH.\n\n" +
			"Install it with Homebrew:\n" +
			"    brew install ffmpeg\n\n" +
			"(No Homebrew? Install it from https://brew.sh first.)"
	case "windows":
		return "ffmpeg was not found on your PATH.\n\n" +
			"Install it with winget:\n" +
			"    winget install ffmpeg\n\n" +
			"or with Chocolatey:\n" +
			"    choco install ffmpeg\n\n" +
			"or download a static build from https://ffmpeg.org/download.html\n" +
			"and add its bin\\ folder to your PATH."
	case "linux":
		return "ffmpeg was not found on your PATH.\n\n" +
			"Install it with your distro's package manager, e.g.:\n" +
			"    sudo apt install ffmpeg        # Debian/Ubuntu\n" +
			"    sudo dnf install ffmpeg        # Fedora\n" +
			"    sudo pacman -S ffmpeg          # Arch\n"
	default:
		return fmt.Sprintf("ffmpeg was not found on your PATH (OS: %s).\n"+
			"Install ffmpeg for your platform from https://ffmpeg.org/download.html", runtime.GOOS)
	}
}
