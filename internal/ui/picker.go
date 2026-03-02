package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/benfourie/fl/internal/jira"
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

type pickerModel struct {
	items    []*jira.Transition
	cursor   int
	chosen   *jira.Transition
	quitting bool
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
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter":
			m.chosen = m.items[m.cursor]
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
	sb.WriteString(headerStyle.Render("Move ticket to:"))
	sb.WriteString("\n\n")

	for i, t := range m.items {
		cursor := "  "
		style := normalStyle
		if i == m.cursor {
			cursor = "▶ "
			style = selectedStyle
		}
		sb.WriteString(fmt.Sprintf("%s%s\n", cursor, style.Render(t.Name)))
	}

	sb.WriteString(dimStyle.Render("\n↑/↓ navigate  enter select  esc cancel"))
	return sb.String()
}

// PickTransition shows an interactive list and returns the chosen transition.
func PickTransition(transitions []*jira.Transition) (*jira.Transition, error) {
	m := pickerModel{items: transitions}

	p := tea.NewProgram(m, tea.WithOutput(os.Stderr))
	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	final := result.(pickerModel)
	if final.chosen == nil {
		return nil, fmt.Errorf("no transition selected")
	}
	return final.chosen, nil
}

// PickTransitionFallback is a plain-text fallback for non-interactive terminals.
func PickTransitionFallback(transitions []*jira.Transition) (*jira.Transition, error) {
	fmt.Println("Available transitions:")
	for i, t := range transitions {
		fmt.Printf("  %d) %s\n", i+1, t.Name)
	}

	var choice int
	fmt.Print("Enter number: ")
	if _, err := fmt.Scan(&choice); err != nil || choice < 1 || choice > len(transitions) {
		return nil, fmt.Errorf("invalid selection")
	}
	return transitions[choice-1], nil
}
