package ui

import (
	"fmt"
	"strings"

	"github.com/benfo/fl/internal/git"
	"github.com/benfo/fl/internal/tracker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── styles ──────────────────────────────────────────────────────────────────

var (
	listHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99"))

	listDividerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240"))

	listCursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			Bold(true)

	listKeyStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	listStatusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	listBranchStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	listActiveBranchStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("82")).
				Bold(true)

	listHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))

	listErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
)

// ── message types ────────────────────────────────────────────────────────────

type itemsLoadedMsg struct {
	items []*tracker.Item
	err   error
}

type gitContextMsg struct {
	currentKey string
	branches   map[string]string
}

// ── list item ────────────────────────────────────────────────────────────────

type listItem struct {
	item       *tracker.Item
	branchName string
	hasBranch  bool
	isCurrent  bool
}

// ── screen ───────────────────────────────────────────────────────────────────

type itemListScreen struct {
	rawItems    []*tracker.Item
	items       []listItem
	gitBranches map[string]string
	currentKey  string
	cursor      int
	loading     bool
	err         error
	client      tracker.Client
	width       int
	height      int
	showHelp    bool
}

// NewItemListScreen creates the item list screen.
func NewItemListScreen(client tracker.Client) *itemListScreen {
	return &itemListScreen{
		client:  client,
		loading: true,
	}
}

func (m itemListScreen) Init() tea.Cmd {
	return tea.Batch(m.loadItemsCmd(), m.loadGitContextCmd())
}

func (m itemListScreen) loadItemsCmd() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		items, err := client.MyOpenItems()
		return itemsLoadedMsg{items: items, err: err}
	}
}

func (m itemListScreen) loadGitContextCmd() tea.Cmd {
	client := m.client
	return func() tea.Msg {
		branches, _ := git.LocalBranches()
		currentKey, _ := git.TicketKeyFromBranch(client.KeyPattern())
		return gitContextMsg{currentKey: currentKey, branches: branches}
	}
}

func (m itemListScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case itemsLoadedMsg:
		m.loading = false
		m.err = msg.err
		m.rawItems = msg.items
		m.recomputeItems()
		m.focusCurrentItem()

	case gitContextMsg:
		m.currentKey = msg.currentKey
		m.gitBranches = msg.branches
		m.recomputeItems()
		m.focusCurrentItem()

	case itemUpdatedMsg:
		// Refresh git context when an item's branch may have changed.
		return m, m.loadGitContextCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "enter", "right":
			if len(m.items) > 0 {
				item := m.items[m.cursor].item
				detail := newItemDetailScreen(item, m.client)
				return m, func() tea.Msg { return pushScreenMsg{screen: detail} }
			}
		case "r":
			m.loading = true
			m.rawItems = nil
			return m, tea.Batch(m.loadItemsCmd(), m.loadGitContextCmd())
		case "q", "ctrl+c":
			return m, tea.Quit
		case "?":
			m.showHelp = !m.showHelp
		}
	}
	return m, nil
}

func (m *itemListScreen) recomputeItems() {
	if m.rawItems == nil {
		return
	}
	m.items = make([]listItem, len(m.rawItems))
	for i, item := range m.rawItems {
		branchName, _ := git.BranchName(item)
		_, hasBranch := m.gitBranches[branchName]
		isCurrent := item.Key == m.currentKey
		m.items[i] = listItem{
			item:       item,
			branchName: branchName,
			hasBranch:  hasBranch,
			isCurrent:  isCurrent,
		}
	}
}

func (m *itemListScreen) focusCurrentItem() {
	if m.currentKey == "" {
		return
	}
	for i, li := range m.items {
		if li.isCurrent {
			m.cursor = i
			return
		}
	}
}

func (m itemListScreen) View() string {
	width := m.width
	if width == 0 {
		width = 80
	}
	divider := listDividerStyle.Render(strings.Repeat("─", width))

	var sb strings.Builder

	// Header
	help := listHelpStyle.Render("r refresh  ? help  q quit")
	header := listHeaderStyle.Render("fl — open items")
	padding := width - lipgloss.Width(header) - lipgloss.Width(help)
	if padding < 1 {
		padding = 1
	}
	sb.WriteString(header + strings.Repeat(" ", padding) + help + "\n")
	sb.WriteString(divider + "\n")

	// Loading / error states
	if m.loading {
		sb.WriteString(listHelpStyle.Render("  Loading…") + "\n")
		return sb.String()
	}
	if m.err != nil {
		sb.WriteString(listErrorStyle.Render(fmt.Sprintf("  Error: %v", m.err)) + "\n")
		return sb.String()
	}
	if len(m.items) == 0 {
		sb.WriteString(listHelpStyle.Render("  No open items.") + "\n")
		return sb.String()
	}

	// Item rows
	for i, li := range m.items {
		cursor := "   "
		keyStyle := listKeyStyle
		if i == m.cursor {
			cursor = " ▶ "
			keyStyle = listCursorStyle
		}

		key := keyStyle.Render(fmt.Sprintf("%-12s", li.item.Key))
		status := listStatusStyle.Render(fmt.Sprintf("[%-12s]", truncate(li.item.Status, 12)))

		// Compute available space for summary
		fixed := len(cursor) + lipgloss.Width(key) + lipgloss.Width(status) + 2
		maxSummary := width - fixed - 12 // leave room for branch indicator
		if maxSummary < 10 {
			maxSummary = 10
		}
		summary := truncate(li.item.Summary, maxSummary)

		// Branch indicator
		branchIndicator := ""
		if li.isCurrent {
			branchIndicator = listActiveBranchStyle.Render("  ⎇ active")
		} else if li.hasBranch {
			branchIndicator = listBranchStyle.Render("  ⎇ local")
		}

		sb.WriteString(fmt.Sprintf("%s%s %s  %s%s\n", cursor, key, status, summary, branchIndicator))
	}

	// Help overlay
	if m.showHelp {
		sb.WriteString("\n" + divider + "\n")
		sb.WriteString(listHelpStyle.Render("  ↑/k up  ↓/j down  enter/→ open  r refresh  q quit  ? close help") + "\n")
	}

	return sb.String()
}
