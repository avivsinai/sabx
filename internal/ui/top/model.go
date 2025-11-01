package top

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/sabx/sabx/internal/sabapi"
)

const refreshInterval = 2 * time.Second

// Run launches the Bubble Tea dashboard.
func Run(ctx context.Context, client *sabapi.Client) error {
	m := model{client: client, historyLimit: 15}
	p := tea.NewProgram(m)
	done := make(chan error, 1)

	go func() {
		done <- p.Start()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

type model struct {
	client       *sabapi.Client
	queue        *sabapi.QueueResponse
	status       *sabapi.StatusResponse
	history      []sabapi.HistorySlot
	err          error
	historyLimit int
}

type dataMsg struct {
	queue   *sabapi.QueueResponse
	status  *sabapi.StatusResponse
	history []sabapi.HistorySlot
	err     error
}

type tickMsg struct{}

func (m model) Init() tea.Cmd {
	return tea.Batch(fetchCmd(m.client, m.historyLimit), tickCmd())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		}
	case dataMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.queue = msg.queue
			m.status = msg.status
			m.history = msg.history
			m.err = nil
		}
		return m, tickCmd()
	case tickMsg:
		return m, fetchCmd(m.client, m.historyLimit)
	}
	return m, nil
}

func (m model) View() string {
	var b strings.Builder
	b.WriteString(" sabx top (press q to quit)\n\n")

	if m.err != nil {
		b.WriteString(fmt.Sprintf(" error: %v\n", m.err))
	}

	if m.status != nil {
		b.WriteString(fmt.Sprintf(" status: paused=%v speed=%sKB/s limit=%sKB/s\n", m.status.Paused, valueOr(ms(m.status.Speed)), valueOr(ms(m.status.SpeedLimit))))
	}

	if m.queue != nil {
		b.WriteString(fmt.Sprintf(" queue: %d items, eta=%s, mbleft=%s\n", len(m.queue.Slots), m.queue.TimeLeft, m.queue.MBLeft))
		b.WriteString(" -------------------------------------------------------------\n")
		for _, slot := range m.queue.Slots {
			b.WriteString(fmt.Sprintf(" %-20s %-8s %-8s %-12s\n", trim(slot.Filename, 20), priorityLabel(slot.Priority), slot.Status, slot.Eta))
		}
	}

	if len(m.history) > 0 {
		b.WriteString("\n recent history:\n")
		for i, slot := range m.history {
			if i >= 5 {
				break
			}
			b.WriteString(fmt.Sprintf(" %-20s %-10s %s\n", trim(slot.Name, 20), slot.Status, slot.Completed))
		}
	}

	return b.String()
}

func fetchCmd(client *sabapi.Client, historyLimit int) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()

		queue, err := client.Queue(ctx, 0, 0, "")
		if err != nil {
			return dataMsg{err: err}
		}
		status, err := client.Status(ctx)
		if err != nil {
			return dataMsg{err: err}
		}
		history, err := client.History(ctx, false, historyLimit)
		if err != nil {
			return dataMsg{queue: queue, status: status, err: err}
		}
		return dataMsg{queue: queue, status: status, history: history.Slots}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(refreshInterval, func(time.Time) tea.Msg { return tickMsg{} })
}

func trim(s string, max int) string {
	if len([]rune(s)) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max-1]) + "…"
}

func priorityLabel(priority string) string {
	switch priority {
	case "2":
		return "force"
	case "1":
		return "high"
	case "0":
		return "normal"
	case "-1":
		return "low"
	default:
		return priority
	}
}

func valueOr(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}

func ms(s string) string {
	if strings.TrimSpace(s) == "" {
		return "0"
	}
	return s
}
