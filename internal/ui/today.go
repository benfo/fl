package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/benfourie/fl/internal/calendar"
	"github.com/benfourie/fl/internal/tracker"
	"github.com/charmbracelet/lipgloss"
)

var (
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99")).
			MarginBottom(1)

	ticketKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			PaddingLeft(1)

	eventTimeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Width(7)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	sectionStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(lipgloss.Color("240")).
			MarginBottom(1).
			PaddingBottom(1)
)

// RenderToday prints the today dashboard to stdout.
func RenderToday(items []*tracker.Item, events []*calendar.Event) error {
	w := os.Stdout

	fmt.Fprintln(w, headerStyle.Render("  fl — today"))
	fmt.Fprintln(w, sectionStyle.Render(renderItems(items)))
	fmt.Fprintln(w, renderEvents(events))

	return nil
}

func renderItems(items []*tracker.Item) string {
	if len(items) == 0 {
		return dimStyle.Render("No open items.")
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Tasks"))
	sb.WriteString("\n")

	for _, t := range items {
		key := ticketKeyStyle.Render(t.Key)
		status := statusStyle.Render(fmt.Sprintf("[%s]", t.Status))
		summary := truncate(t.Summary, 60)
		sb.WriteString(fmt.Sprintf("  %s%s  %s\n", key, status, summary))
	}
	return sb.String()
}

func renderEvents(events []*calendar.Event) string {
	if len(events) == 0 {
		return dimStyle.Render("No calendar events today.")
	}

	var sb strings.Builder
	sb.WriteString(lipgloss.NewStyle().Bold(true).Render("Calendar"))
	sb.WriteString("\n")

	for _, e := range events {
		var label string
		if e.AllDay {
			label = "all day"
		} else {
			label = e.Start.Format("15:04")
		}
		timeStr := eventTimeStyle.Render(label)
		provider := dimStyle.Render(fmt.Sprintf("(%s)", e.Provider))
		title := truncate(e.Title, 55)
		sb.WriteString(fmt.Sprintf("  %s %s %s\n", timeStr, title, provider))
	}
	return sb.String()
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
