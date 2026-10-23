// Package tui renders a live Bubble Tea progress view over an
// engine.Run's event stream.
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/randusr836/flac-mirror/internal/engine"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("203"))
	activeStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("110"))
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("114"))
)

type eventMsg engine.Event
type streamClosedMsg struct{}

func listen(events <-chan engine.Event) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-events
		if !ok {
			return streamClosedMsg{}
		}
		return eventMsg(ev)
	}
}

type Model struct {
	events  <-chan engine.Event
	cancel  context.CancelFunc
	workers int

	totalTranscode, totalCopy, totalDelete int
	doneTranscode, doneCopy, doneDelete    int
	skipped                                int

	inProgress  map[string]time.Time
	transcodeDs []time.Duration

	errs []string

	startTime time.Time
	finished  bool
	quitting  bool

	bar   progress.Model
	width int
}

// New builds the Bubble Tea program that will render progress for a run
// whose events arrive on events. cancel is called if the user interrupts
// the run (q / ctrl+c) before it finishes.
func New(events <-chan engine.Event, cancel context.CancelFunc, workers int) *tea.Program {
	m := Model{
		events:     events,
		cancel:     cancel,
		workers:    workers,
		inProgress: map[string]time.Time{},
		bar:        progress.New(progress.WithDefaultGradient()),
	}
	return tea.NewProgram(m)
}

func (m Model) Init() tea.Cmd {
	return listen(m.events)
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.bar.Width = msg.Width - 4
		if m.bar.Width > 60 {
			m.bar.Width = 60
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			if !m.finished {
				m.cancel()
			}
			m.quitting = true
			return m, tea.Quit
		}
		if m.finished {
			m.quitting = true
			return m, tea.Quit
		}
		return m, nil

	case eventMsg:
		m.apply(engine.Event(msg))
		if msg.Type == engine.EventAllDone {
			m.finished = true
			return m, nil
		}
		return m, listen(m.events)

	case streamClosedMsg:
		m.finished = true
		return m, nil
	}
	return m, nil
}

func (m *Model) apply(ev engine.Event) {
	switch ev.Type {
	case engine.EventPlanReady:
		m.totalTranscode = len(ev.Plan.ToTranscode)
		m.totalCopy = len(ev.Plan.ToCopy)
		m.totalDelete = len(ev.Plan.ToDelete)
		m.skipped = len(ev.Plan.UpToDate)
		m.startTime = time.Now()

	case engine.EventJobStart:
		m.inProgress[ev.Item.RelPath] = time.Now()

	case engine.EventJobDone:
		delete(m.inProgress, ev.Item.RelPath)
		if ev.Err != nil {
			m.errs = append(m.errs, fmt.Sprintf("%s: %v", ev.Item.RelPath, ev.Err))
		}
		switch ev.Item.Kind.String() {
		case "transcode":
			m.doneTranscode++
			m.transcodeDs = append(m.transcodeDs, ev.Duration)
		default:
			m.doneCopy++
		}

	case engine.EventDeleteDone:
		m.doneDelete++
	}
}

func (m Model) total() int   { return m.totalTranscode + m.totalCopy + m.totalDelete }
func (m Model) doneAll() int { return m.doneTranscode + m.doneCopy + m.doneDelete }

// ErrCount returns how many jobs failed, for the caller to check after
// the program exits.
func (m Model) ErrCount() int { return len(m.errs) }

func (m Model) eta() string {
	remaining := m.totalTranscode - m.doneTranscode
	if remaining <= 0 {
		return "0s"
	}
	if len(m.transcodeDs) == 0 {
		return "calculating..."
	}
	var sum time.Duration
	for _, d := range m.transcodeDs {
		sum += d
	}
	avg := sum / time.Duration(len(m.transcodeDs))
	workers := m.workers
	if workers < 1 {
		workers = 1
	}
	eta := time.Duration(int64(avg) * int64(remaining) / int64(workers))
	return eta.Round(time.Second).String()
}

func (m Model) View() string {
	if m.total() == 0 && m.startTime.IsZero() {
		return dimStyle.Render("Scanning library...") + "\n"
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("flac-mirror") + "\n\n")

	total := m.total()
	done := m.doneAll()
	frac := 0.0
	if total > 0 {
		frac = float64(done) / float64(total)
	}
	b.WriteString(m.bar.ViewAs(frac) + "\n\n")

	b.WriteString(fmt.Sprintf(
		"transcode %d/%d   copy %d/%d   delete %d/%d   %s\n",
		m.doneTranscode, m.totalTranscode,
		m.doneCopy, m.totalCopy,
		m.doneDelete, m.totalDelete,
		dimStyle.Render(fmt.Sprintf("(%d already up to date)", m.skipped)),
	))

	if !m.finished {
		b.WriteString(fmt.Sprintf("ETA: %s\n\n", m.eta()))
	} else {
		elapsed := time.Since(m.startTime).Round(time.Second)
		b.WriteString(successStyle.Render(fmt.Sprintf("Done in %s.", elapsed)) + "\n\n")
	}

	if !m.finished && len(m.inProgress) > 0 {
		names := make([]string, 0, len(m.inProgress))
		for name := range m.inProgress {
			names = append(names, name)
		}
		sort.Strings(names)
		max := 8
		for i, name := range names {
			if i >= max {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more\n", len(names)-max)))
				break
			}
			b.WriteString(activeStyle.Render("  ⠿ "+name) + "\n")
		}
		b.WriteString("\n")
	}

	if len(m.errs) > 0 {
		b.WriteString(errorStyle.Render(fmt.Sprintf("%d error(s):", len(m.errs))) + "\n")
		start := 0
		if len(m.errs) > 5 {
			start = len(m.errs) - 5
		}
		for _, e := range m.errs[start:] {
			b.WriteString(errorStyle.Render("  "+e) + "\n")
		}
		b.WriteString("\n")
	}

	if m.finished {
		b.WriteString(dimStyle.Render("Press any key to exit.") + "\n")
	} else {
		b.WriteString(dimStyle.Render("Press q to cancel.") + "\n")
	}

	return b.String()
}
