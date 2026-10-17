// Package engine runs a library.Plan across a worker pool and reports
// progress over a channel of Events.
package engine

import (
	"context"
	"os"
	"sync"
	"time"

	"flacmirror/internal/copyfile"
	"flacmirror/internal/library"
	"flacmirror/internal/transcode"
)

// EventType identifies what a given Event is reporting.
type EventType int

const (
	EventPlanReady EventType = iota
	EventJobStart
	EventJobDone
	EventDeleteDone
	EventAllDone
)

// Event is a single progress update emitted while a Plan is being run.
type Event struct {
	Type     EventType
	Item     library.Item
	Err      error
	Duration time.Duration
	Path     string // for EventDeleteDone
	Plan     library.Plan
}

// Options controls how Run executes a Plan.
type Options struct {
	Workers  int
	Bitrate  string
	DestRoot string
}

// Run executes every job in plan across Options.Workers goroutines and
// deletes every orphaned path in plan.ToDelete, sending an Event for each
// step on events. events is closed when Run returns. Run respects
// ctx cancellation, aborting remaining work.
func Run(ctx context.Context, plan library.Plan, opts Options, events chan<- Event) {
	defer close(events)

	events <- Event{Type: EventPlanReady, Plan: plan}

	// Deletions are cheap; do them up front, sequentially.
	for _, path := range plan.ToDelete {
		if ctx.Err() != nil {
			return
		}
		_ = os.Remove(path)
		events <- Event{Type: EventDeleteDone, Path: path}
	}

	jobs := make(chan library.Item)
	var wg sync.WaitGroup

	workers := opts.Workers
	if workers < 1 {
		workers = 1
	}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				if ctx.Err() != nil {
					return
				}
				runOne(ctx, item, opts.Bitrate, events)
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, item := range plan.ToTranscode {
			select {
			case jobs <- item:
			case <-ctx.Done():
				return
			}
		}
		for _, item := range plan.ToCopy {
			select {
			case jobs <- item:
			case <-ctx.Done():
				return
			}
		}
	}()

	wg.Wait()

	if opts.DestRoot != "" {
		_ = library.PruneEmptyDirs(opts.DestRoot) // best effort
	}

	events <- Event{Type: EventAllDone}
}

func runOne(ctx context.Context, item library.Item, bitrate string, events chan<- Event) {
	events <- Event{Type: EventJobStart, Item: item}
	start := time.Now()

	var err error
	switch item.Kind {
	case library.KindTranscode:
		err = transcode.ToMP3(ctx, item.SourcePath, item.DestPath, bitrate)
	case library.KindPlaylist:
		err = copyfile.Playlist(item.SourcePath, item.DestPath)
	case library.KindCoverArt:
		err = copyfile.Plain(item.SourcePath, item.DestPath)
	}

	events <- Event{Type: EventJobDone, Item: item, Err: err, Duration: time.Since(start)}
}
