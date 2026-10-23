// Command flacmirror mirrors a FLAC music library onto a destination
// (e.g. an SD card) as 320kbps MP3s, transcoding in parallel across CPU
// cores and keeping the destination an exact mirror of the source.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/mattn/go-isatty"

	"github.com/randusr836/flac-mirror/internal/engine"
	"github.com/randusr836/flac-mirror/internal/ffmpegutil"
	"github.com/randusr836/flac-mirror/internal/library"
	"github.com/randusr836/flac-mirror/internal/tui"
)

// version is set at build time via -ldflags "-X main.version=...", e.g. by
// goreleaser from the release tag. Builds that don't set it (e.g. go install)
// fall back to the module version recorded in the binary's build info.
var version = "dev"

func init() {
	if version != "dev" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	workers := flag.Int("j", runtime.NumCPU(), "number of parallel transcode workers")
	bitrate := flag.String("bitrate", "320k", "MP3 CBR bitrate")
	dryRun := flag.Bool("dry-run", false, "show what would change without touching any files")
	yes := flag.Bool("y", false, "skip the confirmation prompt")
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags] <source-dir> <dest-dir>\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Mirrors a FLAC library at <source-dir> onto <dest-dir> as 320kbps MP3s,")
		fmt.Fprintln(os.Stderr, "deleting anything on <dest-dir> with no corresponding source file.")
		fmt.Fprintln(os.Stderr, "\nFlags:")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println("flacmirror " + version)
		return nil
	}

	if flag.NArg() != 2 {
		flag.Usage()
		os.Exit(2)
	}
	source, dest := flag.Arg(0), flag.Arg(1)

	if err := ffmpegutil.Check(); err != nil {
		fmt.Fprintln(os.Stderr, ffmpegutil.InstallInstructions())
		return err
	}

	if info, err := os.Stat(source); err != nil || !info.IsDir() {
		return fmt.Errorf("source %q is not a directory", source)
	}

	fmt.Println("Scanning library...")
	plan, err := library.Build(source, dest)
	if err != nil {
		return fmt.Errorf("building plan: %w", err)
	}

	printSummary(plan)

	if len(plan.ToTranscode)+len(plan.ToCopy)+len(plan.ToDelete) == 0 {
		fmt.Println("Already up to date.")
		return nil
	}

	if *dryRun {
		fmt.Println("\nDry run, no files were changed.")
		return nil
	}

	if err := library.CheckWritable(dest); err != nil {
		return err
	}

	if !*yes && len(plan.ToDelete) > 0 {
		if !confirm(fmt.Sprintf("This will permanently delete %d file(s) from %s. Continue? [y/N] ", len(plan.ToDelete), dest)) {
			fmt.Println("Aborted.")
			return nil
		}
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	events := make(chan engine.Event)
	opts := engine.Options{Workers: *workers, Bitrate: *bitrate, DestRoot: dest}

	go engine.Run(ctx, plan, opts, events)

	var errCount int
	if isatty.IsTerminal(os.Stdout.Fd()) {
		errCount, err = runWithTUI(events, cancel, *workers)
		if err != nil {
			return err
		}
	} else {
		errCount = runPlain(events)
	}

	if errCount > 0 {
		return fmt.Errorf("%d file(s) failed", errCount)
	}
	return nil
}

func printSummary(plan library.Plan) {
	fmt.Printf("%d to transcode, %d to copy, %d up to date, %d to delete\n",
		len(plan.ToTranscode), len(plan.ToCopy), len(plan.UpToDate), len(plan.ToDelete))
}

func confirm(prompt string) bool {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	switch line {
	case "y\n", "Y\n", "y\r\n", "Y\r\n":
		return true
	default:
		return false
	}
}

func runWithTUI(events chan engine.Event, cancel context.CancelFunc, workers int) (int, error) {
	p := tui.New(events, cancel, workers)
	final, err := p.Run()
	if err != nil {
		return 0, fmt.Errorf("running TUI: %w", err)
	}
	return final.(tui.Model).ErrCount(), nil
}

func runPlain(events chan engine.Event) int {
	errCount := 0
	for ev := range events {
		switch ev.Type {
		case engine.EventJobDone:
			if ev.Err != nil {
				errCount++
				fmt.Printf("FAILED  %s: %v\n", ev.Item.RelPath, ev.Err)
			} else {
				fmt.Printf("done    %s\n", ev.Item.RelPath)
			}
		case engine.EventDeleteDone:
			fmt.Printf("deleted %s\n", ev.Path)
		case engine.EventAllDone:
			fmt.Println("All done.")
		}
	}
	return errCount
}
