package ui

import (
	"fmt"
	"strings"

	"github.com/benfo/fl/internal/tracker"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── message types ─────────────────────────────────────────────────────────────

type createDestsLoadedMsg struct {
	dests []*tracker.CreateDest
	err   error
}

type createDoneMsg struct {
	item *tracker.Item
	err  error
}

// ── screen ────────────────────────────────────────────────────────────────────

type createItemScreen struct {
	step         int // 0=summary, 1=desc, 2=dest picker
	summaryInput textinput.Model
	descInput    textarea.Model
	dests        []*tracker.CreateDest
	destCursor   int
	destsLoading bool
	assignToMe   bool
	creating     bool
	statusMsg    string
	statusIsErr  bool
	client       tracker.Client
	width        int
	height       int
}

func newCreateItemScreen(client tracker.Client) *createItemScreen {
	si := textinput.New()
	si.Placeholder = "Item summary…"
	si.CharLimit = 256

	ta := textarea.New()
	ta.Placeholder = "Description (optional)…"
	ta.CharLimit = 0

	return &createItemScreen{
		client:       client,
		summaryInput: si,
		descInput:    ta,
		destsLoading: true,
		assignToMe:   true,
	}
}

func (m createItemScreen) Init() tea.Cmd {
	return tea.Batch(
		m.summaryInput.Focus(),
		m.loadDestsCmd(),
	)
}

func (m createItemScreen) loadDestsCmd() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		dests, err := client.CreateDests()
		return createDestsLoadedMsg{dests: dests, err: err}
	}
}

func (m createItemScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.descInput.SetWidth(msg.Width - 6)
		m.descInput.SetHeight(6)
		return m, nil

	case createDestsLoadedMsg:
		m.destsLoading = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error loading destinations: %v", msg.err)
			m.statusIsErr = true
		} else {
			m.dests = msg.dests
		}
		return m, nil

	case createDoneMsg:
		m.creating = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
			m.statusIsErr = true
			return m, nil
		}
		item := msg.item
		return m, tea.Batch(
			func() tea.Msg { return popScreenMsg{} },
			func() tea.Msg { return itemUpdatedMsg{item: item} },
		)

	case tea.KeyMsg:
		switch m.step {
		case 0:
			return m.updateStepSummary(msg)
		case 1:
			return m.updateStepDesc(msg)
		case 2:
			return m.updateStepDest(msg)
		}
	}
	return m, nil
}

func (m createItemScreen) updateStepSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		if strings.TrimSpace(m.summaryInput.Value()) != "" {
			m.step = 1
			m.descInput.SetValue("")
			return m, m.descInput.Focus()
		}
		return m, nil
	case "esc":
		return m, func() tea.Msg { return popScreenMsg{} }
	}
	var cmd tea.Cmd
	m.summaryInput, cmd = m.summaryInput.Update(msg)
	return m, cmd
}

func (m createItemScreen) updateStepDesc(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+s":
		// Advance keeping description.
		m.step = 2
		m.destCursor = 0
		m.statusMsg = ""
		return m, nil
	case "tab":
		// Skip description.
		m.descInput.SetValue("")
		m.step = 2
		m.destCursor = 0
		m.statusMsg = ""
		return m, nil
	case "esc":
		m.step = 0
		return m, m.summaryInput.Focus()
	}
	var cmd tea.Cmd
	m.descInput, cmd = m.descInput.Update(msg)
	return m, cmd
}

func (m createItemScreen) updateStepDest(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.destCursor > 0 {
			m.destCursor--
		}
	case "down", "j":
		if m.destCursor < len(m.dests)-1 {
			m.destCursor++
		}
	case "a":
		m.assignToMe = !m.assignToMe
	case "enter":
		if m.creating {
			return m, nil
		}
		if m.destsLoading {
			m.statusMsg = "Still loading destinations…"
			m.statusIsErr = false
			return m, nil
		}
		if len(m.dests) == 0 {
			m.statusMsg = "No destinations available."
			m.statusIsErr = true
			return m, nil
		}
		m.creating = true
		m.statusMsg = ""
		dest := m.dests[m.destCursor]
		summary := strings.TrimSpace(m.summaryInput.Value())
		description := m.descInput.Value()
		assignToMe := m.assignToMe
		client := m.client
		return m, func() tea.Msg {
			item, err := client.CreateItem(dest.ID, summary, description)
			if err != nil {
				return createDoneMsg{err: err}
			}
			if assignToMe {
				_ = client.AssignToMe(item.Key)
			}
			return createDoneMsg{item: item}
		}
	case "esc":
		m.step = 1
		return m, m.descInput.Focus()
	}
	return m, nil
}

func (m createItemScreen) View() string {
	width := m.width
	if width == 0 {
		width = 80
	}
	divider := detailDivStyle.Render(strings.Repeat("─", width))

	var sb strings.Builder

	// Header
	sb.WriteString(detailHeaderStyle.Render("Create item") + "\n")
	sb.WriteString(divider + "\n")

	switch m.step {
	case 0:
		sb.WriteString("  " + detailLabelStyle.Render("Summary") + "\n")
		si := m.summaryInput
		si.Width = width - 4
		sb.WriteString("  " + si.View() + "\n\n")
		sb.WriteString("  " + detailHelpStyle.Render("Enter next  Esc cancel") + "\n")

	case 1:
		sb.WriteString("  " + detailLabelStyle.Render("Summary") + "\n")
		sb.WriteString("  " + detailDescStyle.Render(m.summaryInput.Value()) + "\n\n")
		sb.WriteString("  " + detailLabelStyle.Render("Description") + "\n")
		sb.WriteString(detailPanelStyle.Width(width-4).Render(m.descInput.View()) + "\n\n")
		sb.WriteString("  " + detailHelpStyle.Render("Ctrl+S next  Tab skip desc  Esc back") + "\n")

	case 2:
		sb.WriteString("  " + detailLabelStyle.Render("Summary") + "\n")
		sb.WriteString("  " + detailDescStyle.Render(m.summaryInput.Value()) + "\n\n")

		assignLabel := "yes"
		if !m.assignToMe {
			assignLabel = "no"
		}

		if m.destsLoading {
			sb.WriteString("  " + detailHelpStyle.Render("Loading destinations…") + "\n\n")
		} else if len(m.dests) == 0 {
			sb.WriteString("  " + detailErrStyle.Render("No destinations available.") + "\n\n")
		} else {
			sb.WriteString("  " + detailLabelStyle.Render("Destination") + "\n")
			for i, dest := range m.dests {
				cursor := "  "
				style := detailSubStyle
				if i == m.destCursor {
					cursor = "▶ "
					style = detailSubSelStyle
				}
				sb.WriteString("  " + cursor + style.Render(truncate(dest.Label, width-8)) + "\n")
			}
			sb.WriteString("\n")
		}

		if m.creating {
			sb.WriteString("  " + detailHelpStyle.Render("Creating…") + "\n")
		} else {
			sb.WriteString("  " + detailHelpStyle.Render(
				fmt.Sprintf("a assign: %s  Enter create  Esc back", lipgloss.NewStyle().Bold(true).Render(assignLabel)),
			) + "\n")
		}
	}

	// Status message
	if m.statusMsg != "" {
		sb.WriteString("\n")
		if m.statusIsErr {
			sb.WriteString("  " + detailErrStyle.Render(m.statusMsg) + "\n")
		} else {
			sb.WriteString("  " + detailOkStyle.Render(m.statusMsg) + "\n")
		}
	}

	return sb.String()
}
