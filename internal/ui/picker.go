package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/benfo/flow-cli/internal/tracker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			Bold(true)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)

// pickerModel is a generic bubbletea list picker that works with any []string.
type pickerModel struct {
	prompt   string
	options  []string
	cursor   int
	chosen   int // -1 = cancelled
	quitting bool
}

func newPickerModel(prompt string, options []string) pickerModel {
	return pickerModel{prompt: prompt, options: options, chosen: -1}
}

func (m pickerModel) Init() tea.Cmd { return nil }

func (m pickerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case "enter":
			m.chosen = m.cursor
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m pickerModel) View() string {
	if m.quitting {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(headerStyle.Render(m.prompt))
	sb.WriteString("\n\n")

	for i, opt := range m.options {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "▶ "
			style = selectedStyle
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(opt)))
	}

	sb.WriteString(dimStyle.Render("\n↑/↓ navigate  enter select  esc cancel"))
	return sb.String()
}

// pickIndex runs the TUI picker and returns the selected index, or -1 if cancelled.
func pickIndex(prompt string, options []string) (int, error) {
	m := newPickerModel(prompt, options)
	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return -1, err
	}
	final := result.(pickerModel)
	if final.chosen == -1 {
		return -1, fmt.Errorf("cancelled")
	}
	return final.chosen, nil
}

// pickIndexFallback is a plain-text fallback for non-interactive terminals.
func pickIndexFallback(prompt string, options []string) (int, error) {
	fmt.Println(prompt)
	for i, opt := range options {
		fmt.Printf("  %d) %s\n", i+1, opt)
	}
	var choice int
	fmt.Print("Enter number: ")
	if _, err := fmt.Scan(&choice); err != nil || choice < 1 || choice > len(options) {
		return -1, fmt.Errorf("invalid selection")
	}
	return choice - 1, nil
}

// PickTransition shows an interactive list and returns the chosen transition.
func PickTransition(transitions []*tracker.Transition) (*tracker.Transition, error) {
	names := make([]string, len(transitions))
	for i, t := range transitions {
		names[i] = t.Name
	}
	idx, err := pickIndex("Move to:", names)
	if err != nil {
		return nil, err
	}
	return transitions[idx], nil
}

// PickTransitionFallback is a plain-text fallback for non-interactive terminals.
func PickTransitionFallback(transitions []*tracker.Transition) (*tracker.Transition, error) {
	names := make([]string, len(transitions))
	for i, t := range transitions {
		names[i] = t.Name
	}
	idx, err := pickIndexFallback("Available transitions:", names)
	if err != nil {
		return nil, err
	}
	return transitions[idx], nil
}

// PickCreateDest shows an interactive list and returns the chosen create destination.
func PickCreateDest(dests []*tracker.CreateDest) (*tracker.CreateDest, error) {
	labels := make([]string, len(dests))
	for i, d := range dests {
		labels[i] = d.Label
	}
	idx, err := pickIndex("Create in:", labels)
	if err != nil {
		return nil, err
	}
	return dests[idx], nil
}

// PickCreateDestFallback is a plain-text fallback for non-interactive terminals.
func PickCreateDestFallback(dests []*tracker.CreateDest) (*tracker.CreateDest, error) {
	labels := make([]string, len(dests))
	for i, d := range dests {
		labels[i] = d.Label
	}
	idx, err := pickIndexFallback("Create in:", labels)
	if err != nil {
		return nil, err
	}
	return dests[idx], nil
}
