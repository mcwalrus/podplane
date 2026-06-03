// Podplane <https://podplane.dev>
// Copyright The Podplane Authors
// SPDX-License-Identifier: Apache-2.0

package tui

import (
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

const taskProgressBarWidth = 20
const taskProgressOverallBarWidth = 28

// TaskProgressEventType describes a lifecycle transition for one task progress
// row. Callers normally do not need to construct these values directly; use the
// TaskProgress helper methods passed to RunTaskProgress instead.
type TaskProgressEventType string

const (
	// TaskProgressStarted marks an item as actively running. Started items show
	// elapsed time and, when Expected is set, an expectation bar until that
	// duration is exceeded.
	TaskProgressStarted TaskProgressEventType = "started"

	// TaskProgressDone marks an item as complete. Done items count toward the
	// overall completion total and contribute their full expected duration to the
	// overall expectation bar.
	TaskProgressDone TaskProgressEventType = "done"

	// TaskProgressOmitted removes an item from the progress UI when the caller
	// determines that the phase does not apply to this run.
	TaskProgressOmitted TaskProgressEventType = "omitted"

	// TaskProgressFailed marks an item as failed and displays the event error in
	// the progress UI.
	TaskProgressFailed TaskProgressEventType = "failed"

	// TaskProgressSkipped marks an item as already satisfied, such as a reused
	// local server or existing VM image.
	TaskProgressSkipped TaskProgressEventType = "skipped"

	// TaskProgressInfo updates an item's message without changing its lifecycle
	// state.
	TaskProgressInfo TaskProgressEventType = "info"
)

// TaskProgressItem describes one row in a task progress UI. Items are rendered
// in the order provided to RunTaskProgress, and their Expected durations are
// summed to produce the overall expected-time bar.
type TaskProgressItem struct {
	// Key is the stable identifier used by events to update this row.
	Key string

	// Name is the user-facing row label shown in the progress UI.
	Name string

	// Exclude removes this item from the progress UI entirely. Excluded items do
	// not render and do not contribute to the overall expected-time bar.
	Exclude bool

	// Success is the message shown when this item completes successfully and the
	// completion event does not provide a more specific message.
	Success string

	// Expected is the usual duration for this task. It is used only to set user
	// expectations; it is not treated as actual work completion.
	Expected time.Duration

	// Timeout is the maximum wait duration for this task when the caller has one.
	// It is displayed as a hint after Expected is exceeded.
	Timeout time.Duration
}

// TaskProgressEvent reports progress for one task progress item. The progress
// UI is intentionally event-driven so long-running operations can remain owned
// by their domain package while the command layer decides whether to render a
// TTY dashboard or line-oriented fallback.
type TaskProgressEvent struct {
	// Type is the lifecycle transition or message update being reported.
	Type TaskProgressEventType

	// Key identifies the item this event updates.
	Key string

	// Name can set or replace the item label when the event creates an item not
	// present in the initial item list.
	Name string

	// Message is an optional user-facing detail rendered next to the item.
	Message string

	// Err is displayed for failed items.
	Err error
}

// TaskProgress emits task progress events. Its methods are nil-safe so domain
// code can report progress unconditionally; callers that do not need a progress
// UI can leave the progress value nil.
type TaskProgress func(TaskProgressEvent)

// Started emits a task-started event for key. Timing metadata comes from the
// TaskProgressItem supplied to RunTaskProgress.
func (p TaskProgress) Started(key, name, message string) {
	if p == nil {
		return
	}
	p(TaskProgressEvent{Type: TaskProgressStarted, Key: key, Name: name, Message: message})
}

// Done emits a task-done event for key with an optional completion message.
func (p TaskProgress) Done(key, name, message string) {
	if p == nil {
		return
	}
	p(TaskProgressEvent{Type: TaskProgressDone, Key: key, Name: name, Message: message})
}

// Omitted emits a task-omitted event for key. Omitted items are hidden and do
// not contribute to the overall expected-time summary.
func (p TaskProgress) Omitted(key, name string) {
	if p == nil {
		return
	}
	p(TaskProgressEvent{Type: TaskProgressOmitted, Key: key, Name: name})
}

// Skipped emits a task-skipped event for key. Skipped items are rendered as
// satisfied and count as complete in the overall summary.
func (p TaskProgress) Skipped(key, name, message string) {
	if p == nil {
		return
	}
	p(TaskProgressEvent{Type: TaskProgressSkipped, Key: key, Name: name, Message: message})
}

// Failed emits a task-failed event for key and displays err in the progress UI.
func (p TaskProgress) Failed(key, name string, err error) {
	if p == nil {
		return
	}
	p(TaskProgressEvent{Type: TaskProgressFailed, Key: key, Name: name, Err: err})
}

// RunTaskProgress renders sequential task progress while run executes. The UI
// includes an overall expected-time bar plus per-item rows; both are expectation
// indicators, not exact work-completion percentages. It falls back to
// line-oriented output when stdout is not a terminal.
func RunTaskProgress(title string, items []TaskProgressItem, run func(TaskProgress) error) error {
	items = includedTaskProgressItems(items)
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return runTextTaskProgress(items, run)
	}

	events := make(chan TaskProgressEvent, 64)
	m := taskProgressModel{
		title:  title,
		run:    run,
		events: events,
		items:  items,
		rows:   map[string]taskProgressRow{},
	}
	for _, item := range items {
		m.rows[item.Key] = taskProgressRow{item: item, status: "pending"}
	}
	finalModel, err := tea.NewProgram(m).Run()
	if err != nil {
		return fmt.Errorf("error running task progress: %w", err)
	}
	if m, ok := finalModel.(taskProgressModel); ok {
		return m.err
	}
	return nil
}

// includedTaskProgressItems returns only items that should participate in the
// progress UI and expected-time totals.
func includedTaskProgressItems(items []TaskProgressItem) []TaskProgressItem {
	included := make([]TaskProgressItem, 0, len(items))
	for _, item := range items {
		if item.Exclude {
			continue
		}
		included = append(included, item)
	}
	return included
}

type taskProgressRow struct {
	item      TaskProgressItem
	status    string
	message   string
	seen      bool
	omitted   bool
	startedAt time.Time
	doneAt    time.Time
	err       error
}

type taskProgressModel struct {
	title        string
	run          func(TaskProgress) error
	events       chan TaskProgressEvent
	items        []TaskProgressItem
	rows         map[string]taskProgressRow
	err          error
	doneReceived bool
	closed       bool
	width        int
}

type taskProgressEventMsg TaskProgressEvent

type taskProgressDoneMsg struct {
	err error
}

type taskProgressClosedMsg struct{}

type taskProgressTickMsg struct{}

// Init starts the background task, event reader, and repaint ticker.
func (m taskProgressModel) Init() tea.Cmd {
	return tea.Batch(m.runCommand(), m.waitForEvent(), taskProgressTick())
}

// runCommand executes the caller's work and forwards emitted progress events
// into the Bubble Tea update loop.
func (m taskProgressModel) runCommand() tea.Cmd {
	return func() tea.Msg {
		progress := TaskProgress(func(event TaskProgressEvent) {
			select {
			case m.events <- event:
			default:
			}
		})
		err := m.run(progress)
		close(m.events)
		return taskProgressDoneMsg{err: err}
	}
}

// waitForEvent waits for the next progress event or reports that the event
// channel has closed.
func (m taskProgressModel) waitForEvent() tea.Cmd {
	return func() tea.Msg {
		event, ok := <-m.events
		if !ok {
			return taskProgressClosedMsg{}
		}
		return taskProgressEventMsg(event)
	}
}

// taskProgressTick schedules periodic repaints so elapsed-time bars update
// even when no new task events arrive.
func taskProgressTick() tea.Cmd {
	return tea.Tick(200*time.Millisecond, func(time.Time) tea.Msg { return taskProgressTickMsg{} })
}

// Update handles key input, task lifecycle events, repaint ticks, and task
// completion signals.
func (m taskProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.err = fmt.Errorf("task progress cancelled")
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case taskProgressEventMsg:
		m.applyEvent(TaskProgressEvent(msg))
		return m, m.waitForEvent()
	case taskProgressTickMsg:
		if m.doneReceived && m.closed {
			return m, tea.Quit
		}
		return m, taskProgressTick()
	case taskProgressDoneMsg:
		m.doneReceived = true
		m.err = msg.err
		if m.closed {
			return m, tea.Quit
		}
		return m, nil
	case taskProgressClosedMsg:
		m.closed = true
		if m.doneReceived {
			return m, tea.Quit
		}
		return m, nil
	}
	return m, nil
}

// applyEvent updates the row state for one progress event, creating a row when
// callers emit an event for a key that was not in the initial item list.
func (m *taskProgressModel) applyEvent(event TaskProgressEvent) {
	if event.Key == "" {
		return
	}
	row, ok := m.rows[event.Key]
	if !ok {
		row = taskProgressRow{item: TaskProgressItem{Key: event.Key, Name: event.Name}, status: "pending"}
		m.items = append(m.items, row.item)
	}
	row.seen = true
	switch event.Type {
	case TaskProgressStarted:
		row.status = "running"
		row.startedAt = time.Now()
		row.doneAt = time.Time{}
		row.err = nil
		if event.Message != "" {
			row.message = event.Message
		}
	case TaskProgressDone:
		row.status = "done"
		if row.startedAt.IsZero() {
			row.startedAt = time.Now()
		}
		row.doneAt = time.Now()
		row.message = event.Message
	case TaskProgressOmitted:
		row.omitted = true
	case TaskProgressFailed:
		row.status = "failed"
		row.err = event.Err
		row.doneAt = time.Now()
		if event.Message != "" {
			row.message = event.Message
		}
	case TaskProgressSkipped:
		row.status = "skipped"
		row.doneAt = time.Now()
		if event.Message != "" {
			row.message = event.Message
		}
	case TaskProgressInfo:
		// Message-only update.
		if event.Message != "" {
			row.message = event.Message
		}
	}
	m.rows[event.Key] = row
}

// View renders the task progress dashboard.
func (m taskProgressModel) View() string {
	var b strings.Builder
	titleStyle := lipgloss.NewStyle().Background(ColorPrimary).Foreground(ColorWhite).Padding(0, 1)
	faintStyle := lipgloss.NewStyle().Faint(true)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff5f87"))

	b.WriteString("\n")
	b.WriteString(titleStyle.Render(m.title))
	b.WriteString("\n\n")
	current, total, complete, count, overExpected := m.overallProgress()
	if total > 0 {
		label := fmt.Sprintf("expected ~%s", formatDuration(total))
		if complete == count {
			label = fmt.Sprintf("took %s", formatDuration(m.elapsed()))
		}
		if overExpected {
			label = "taking longer than usual"
		}
		b.WriteString(m.truncateLine(fmt.Sprintf("Overall  %s  %s · %d/%d complete", renderBar(int64(current), int64(total), taskProgressOverallBarWidth), label, complete, count)))
		b.WriteString("\n\n")
	}
	visible := m.visibleRows()
	for _, row := range visible {
		line := m.truncateLine(renderTaskProgressRow(row))
		if row.status == "failed" {
			line = errorStyle.Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
		if extra := taskProgressExtra(row); extra != "" {
			b.WriteString(faintStyle.Render(m.truncateLine(extra)))
			b.WriteString("\n")
		}
	}
	return b.String()
}

// visibleRows returns rows that should be rendered. Rows that have not emitted
// any event yet are shown only while they are still ahead of the current flow;
// this hides optional phases skipped by the caller, such as VM image creation
// when starting an existing VM.
func (m taskProgressModel) visibleRows() []taskProgressRow {
	lastSeen := -1
	for i, item := range m.items {
		row := m.rows[item.Key]
		if row.seen && !row.omitted {
			lastSeen = i
		}
	}
	if lastSeen < 0 {
		return nil
	}
	rows := make([]taskProgressRow, 0, len(m.items))
	for i, item := range m.items {
		row := m.rows[item.Key]
		if row.omitted {
			continue
		}
		if row.seen || i > lastSeen {
			rows = append(rows, row)
		}
	}
	return rows
}

// truncateLine prevents long row messages from wrapping and leaving stale
// fragments behind during Bubble Tea repaint cycles.
func (m taskProgressModel) truncateLine(line string) string {
	if m.width <= 0 {
		return line
	}
	width := m.width - 1
	if width <= 0 {
		return ""
	}
	runes := []rune(line)
	if len(runes) <= width {
		return line
	}
	if width == 1 {
		return "…"
	}
	return string(runes[:width-1]) + "…"
}

// overallProgress returns the expected-time progress totals, completion counts,
// and whether the currently running task has exceeded its expected duration.
func (m taskProgressModel) overallProgress() (time.Duration, time.Duration, int, int, bool) {
	var current time.Duration
	var total time.Duration
	var complete int
	var tracked int
	var runningOverExpected bool
	allComplete := true
	for _, row := range m.visibleRows() {
		if row.item.Expected <= 0 {
			continue
		}
		tracked++
		total += row.item.Expected
		switch row.status {
		case "done", "skipped":
			complete++
			current += row.item.Expected
		case "running":
			allComplete = false
			elapsed := time.Since(row.startedAt)
			if elapsed > row.item.Expected {
				runningOverExpected = true
				elapsed = row.item.Expected
			}
			current += elapsed
		default:
			allComplete = false
		}
	}
	if total > 0 && current >= total && !allComplete {
		current = total - time.Nanosecond
	}
	return current, total, complete, tracked, runningOverExpected
}

// elapsed returns the wall-clock duration between the first started task and
// the last completed task currently visible in the progress UI.
func (m taskProgressModel) elapsed() time.Duration {
	var start time.Time
	var end time.Time
	for _, row := range m.visibleRows() {
		if !row.startedAt.IsZero() && (start.IsZero() || row.startedAt.Before(start)) {
			start = row.startedAt
		}
		if !row.doneAt.IsZero() && row.doneAt.After(end) {
			end = row.doneAt
		}
	}
	if start.IsZero() || end.IsZero() {
		return 0
	}
	return end.Sub(start)
}

// renderTaskProgressRow renders one task progress row.
func renderTaskProgressRow(row taskProgressRow) string {
	marker := "…"
	state := "waiting"
	suffix := row.message
	switch row.status {
	case "running":
		marker = "⟳"
		state = formatDuration(time.Since(row.startedAt))
		if row.item.Expected > 0 {
			elapsed := time.Since(row.startedAt)
			if elapsed <= row.item.Expected {
				suffix = fmt.Sprintf("expected ~%s  %s", formatDuration(row.item.Expected), renderBar(int64(elapsed), int64(row.item.Expected), taskProgressBarWidth))
			} else {
				suffix = "taking longer than usual"
			}
		}
	case "done":
		marker = "✓"
		state = "done"
		if !row.startedAt.IsZero() && !row.doneAt.IsZero() {
			state = formatDuration(row.doneAt.Sub(row.startedAt))
		}
		if suffix == "" {
			suffix = row.item.Success
		}
	case "failed":
		marker = "✗"
		state = "failed"
		if row.err != nil && suffix == "" {
			suffix = row.err.Error()
		}
	case "skipped":
		marker = "✓"
		state = "ready"
	}
	if suffix == "" {
		return fmt.Sprintf("%s %-26s %s", marker, row.item.Name, state)
	}
	return fmt.Sprintf("%s %-26s %-8s %s", marker, row.item.Name, state, suffix)
}

// taskProgressExtra renders secondary guidance for a row when a task is taking
// longer than its expected duration.
func taskProgressExtra(row taskProgressRow) string {
	if row.status != "running" || row.item.Timeout <= 0 || row.startedAt.IsZero() {
		return ""
	}
	elapsed := time.Since(row.startedAt)
	if row.item.Expected <= 0 || elapsed <= row.item.Expected {
		return ""
	}
	return fmt.Sprintf("   still waiting; timeout %s", formatDuration(row.item.Timeout))
}

// formatDuration renders short human-readable durations for progress rows.
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	minutes := int(d / time.Minute)
	seconds := int((d % time.Minute) / time.Second)
	if seconds == 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return fmt.Sprintf("%dm%02ds", minutes, seconds)
}

// runTextTaskProgress is the non-TTY fallback for task progress.
func runTextTaskProgress(items []TaskProgressItem, run func(TaskProgress) error) error {
	successByKey := map[string]string{}
	for _, item := range items {
		if item.Success != "" {
			successByKey[item.Key] = item.Success
		}
	}
	return run(TaskProgress(func(event TaskProgressEvent) {
		switch event.Type {
		case TaskProgressStarted:
			fmt.Fprintf(os.Stdout, "%s...\n", event.Name)
		case TaskProgressDone:
			message := event.Message
			if message == "" {
				message = successByKey[event.Key]
			}
			if message == "" {
				message = "done"
			}
			fmt.Fprintf(os.Stdout, "✓ %s %s\n", event.Name, message)
		case TaskProgressSkipped:
			if event.Message != "" {
				fmt.Fprintf(os.Stdout, "✓ %s %s\n", event.Name, event.Message)
			}
		case TaskProgressOmitted:
			// Omitted items intentionally produce no non-TTY output.
		case TaskProgressFailed:
			if event.Err != nil {
				fmt.Fprintf(os.Stdout, "❌ %s failed: %v\n", event.Name, event.Err)
			}
		case TaskProgressInfo:
			if event.Message != "" {
				fmt.Fprintln(os.Stdout, event.Message)
			}
		}
	}))
}
