package tui

import (
	"errors"
	"testing"
	"time"

	"github.com/randusr836/flac-mirror/internal/engine"
	"github.com/randusr836/flac-mirror/internal/library"
)

func newTestModel(workers int) Model {
	return Model{
		workers:    workers,
		inProgress: map[string]time.Time{},
	}
}

func TestModelApplyPlanReady(t *testing.T) {
	m := newTestModel(2)
	m.apply(engine.Event{
		Type: engine.EventPlanReady,
		Plan: library.Plan{
			ToTranscode: make([]library.Item, 3),
			ToCopy:      make([]library.Item, 2),
			UpToDate:    make([]library.Item, 5),
			ToDelete:    []string{"a", "b"},
		},
	})

	if m.total() != 7 { // 3 transcode + 2 copy + 2 delete
		t.Errorf("total() = %d, want 7", m.total())
	}
	if m.skipped != 5 {
		t.Errorf("skipped = %d, want 5", m.skipped)
	}
	if m.startTime.IsZero() {
		t.Error("expected startTime to be set")
	}
}

func TestModelApplyJobLifecycle(t *testing.T) {
	m := newTestModel(1)
	item := library.Item{RelPath: "Artist/Album/01 Track.mp3", Kind: library.KindTranscode}

	m.apply(engine.Event{Type: engine.EventJobStart, Item: item})
	if _, inProgress := m.inProgress[item.RelPath]; !inProgress {
		t.Fatal("expected item to be tracked as in-progress after EventJobStart")
	}

	m.apply(engine.Event{Type: engine.EventJobDone, Item: item, Duration: 2 * time.Second})
	if _, stillInProgress := m.inProgress[item.RelPath]; stillInProgress {
		t.Error("expected item to be removed from in-progress after EventJobDone")
	}
	if m.doneTranscode != 1 {
		t.Errorf("doneTranscode = %d, want 1", m.doneTranscode)
	}
	if len(m.transcodeDs) != 1 || m.transcodeDs[0] != 2*time.Second {
		t.Errorf("transcodeDs = %v, want [2s]", m.transcodeDs)
	}
	if m.ErrCount() != 0 {
		t.Errorf("ErrCount() = %d, want 0", m.ErrCount())
	}
}

func TestModelApplyJobDoneCopyKind(t *testing.T) {
	m := newTestModel(1)
	item := library.Item{RelPath: "cover.jpg", Kind: library.KindCoverArt}

	m.apply(engine.Event{Type: engine.EventJobDone, Item: item})
	if m.doneCopy != 1 {
		t.Errorf("doneCopy = %d, want 1", m.doneCopy)
	}
	if m.doneTranscode != 0 {
		t.Errorf("doneTranscode = %d, want 0", m.doneTranscode)
	}
}

func TestModelApplyJobDoneError(t *testing.T) {
	m := newTestModel(1)
	item := library.Item{RelPath: "broken.flac", Kind: library.KindTranscode}

	m.apply(engine.Event{Type: engine.EventJobDone, Item: item, Err: errors.New("boom")})
	if m.ErrCount() != 1 {
		t.Errorf("ErrCount() = %d, want 1", m.ErrCount())
	}
}

func TestModelApplyDeleteDone(t *testing.T) {
	m := newTestModel(1)
	m.apply(engine.Event{Type: engine.EventDeleteDone, Path: "/some/path"})
	m.apply(engine.Event{Type: engine.EventDeleteDone, Path: "/other/path"})
	if m.doneDelete != 2 {
		t.Errorf("doneDelete = %d, want 2", m.doneDelete)
	}
}

func TestEta(t *testing.T) {
	t.Run("nothing remaining", func(t *testing.T) {
		m := newTestModel(4)
		m.totalTranscode, m.doneTranscode = 5, 5
		if got := m.eta(); got != "0s" {
			t.Errorf("eta() = %q, want %q", got, "0s")
		}
	})

	t.Run("no samples yet", func(t *testing.T) {
		m := newTestModel(4)
		m.totalTranscode, m.doneTranscode = 5, 1
		if got := m.eta(); got != "calculating..." {
			t.Errorf("eta() = %q, want %q", got, "calculating...")
		}
	})

	t.Run("estimates from average duration divided across workers", func(t *testing.T) {
		m := newTestModel(2)
		m.totalTranscode, m.doneTranscode = 5, 1 // 4 remaining
		m.transcodeDs = []time.Duration{10 * time.Second}
		// avg=10s, remaining=4, workers=2 -> 10*4/2 = 20s
		if got := m.eta(); got != "20s" {
			t.Errorf("eta() = %q, want %q", got, "20s")
		}
	})

	t.Run("zero workers treated as one", func(t *testing.T) {
		m := newTestModel(0)
		m.totalTranscode, m.doneTranscode = 2, 1
		m.transcodeDs = []time.Duration{5 * time.Second}
		if got := m.eta(); got != "5s" {
			t.Errorf("eta() = %q, want %q", got, "5s")
		}
	})
}

func TestTotalAndDoneAll(t *testing.T) {
	m := newTestModel(1)
	m.totalTranscode, m.totalCopy, m.totalDelete = 3, 2, 1
	m.doneTranscode, m.doneCopy, m.doneDelete = 1, 1, 1

	if m.total() != 6 {
		t.Errorf("total() = %d, want 6", m.total())
	}
	if m.doneAll() != 3 {
		t.Errorf("doneAll() = %d, want 3", m.doneAll())
	}
}
