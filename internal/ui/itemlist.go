package ui

import (
	"fmt"
	"strings"

	"github.com/benfo/fl/internal/git"
	"github.com/benfo/fl/internal/tracker"
	"github.com/charmbracelet/bubbles/textinput"
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

// ── list view modes ───────────────────────────────────────────────────────────

type listViewMode int

const (
	listViewMine       listViewMode = iota
	listViewUnassigned listViewMode = iota
	listViewSearch     listViewMode = iota
)

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

	listView    listViewMode
	searchInput textinput.Model
	searchEntry bool   // true while "/" is active
	searchQuery string // last committed query
}

// NewItemListScreen creates the item list screen.
func NewItemListScreen(client tracker.Client) *itemListScreen {
	si := textinput.New()
	si.Placeholder = "Search…"
	si.CharLimit = 200

	return &itemListScreen{
		client:      client,
		loading:     true,
		searchInput: si,
	}
}

func (m itemListScreen) Init() tea.Cmd {
	return tea.Batch(m.loadItemsCmd(), m.loadGitContextCmd())
}

func (m itemListScreen) loadItemsCmd() tea.Cmd {
	client := m.client
	view := m.listView
	query := m.searchQuery
	return func() tea.Msg {
		var items []*tracker.Item
		var err error
		switch view {
		case listViewMine:
			items, err = client.MyOpenItems()
		case listViewUnassigned:
			items, err = client.UnassignedItems()
		case listViewSearch:
			items, err = client.SearchItems(query)
		}
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
		// Refresh the list and git context after an item change.
		m.loading = true
		m.rawItems = nil
		return m, tea.Batch(m.loadItemsCmd(), m.loadGitContextCmd())

	case tea.KeyMsg:
		// Search entry mode — route all keys to the textinput.
		if m.searchEntry {
			switch msg.String() {
			case "enter":
				query := strings.TrimSpace(m.searchInput.Value())
				m.searchEntry = false
				if query != "" {
					m.searchQuery = query
					m.listView = listViewSearch
					m.cursor = 0
					m.loading = true
					m.rawItems = nil
					return m, m.loadItemsCmd()
				}
				return m, nil
			case "esc":
				m.searchEntry = false
				m.searchInput.SetValue("")
				return m, nil
			}
			var cmd tea.Cmd
			m.searchInput, cmd = m.searchInput.Update(msg)
			return m, cmd
		}

		// Normal mode key handling.
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
		case "tab":
			// Toggle mine ↔ unassigned.
			if m.listView == listViewMine {
				m.listView = listViewUnassigned
			} else {
				m.listView = listViewMine
			}
			m.cursor = 0
			m.loading = true
			m.rawItems = nil
			return m, m.loadItemsCmd()
		case "/":
			m.searchEntry = true
			m.searchInput.SetValue("")
			return m, m.searchInput.Focus()
		case "c":
			screen := newCreateItemScreen(m.client)
			return m, func() tea.Msg { return pushScreenMsg{screen: screen} }
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
	var title, helpText string
	switch m.listView {
	case listViewMine:
		title = "fl — my items"
		helpText = "tab unassigned  / search  c create  r refresh  ? help  q quit"
	case listViewUnassigned:
		title = "fl — unassigned"
		helpText = "tab mine  / search  c create  r refresh  ? help  q quit"
	case listViewSearch:
		title = fmt.Sprintf("fl — search: %q", m.searchQuery)
		helpText = "tab mine  / search  c create  r refresh  ? help  q quit"
	}
	help := listHelpStyle.Render(helpText)
	header := listHeaderStyle.Render(title)
	padding := width - lipgloss.Width(header) - lipgloss.Width(help)
	if padding < 1 {
		padding = 1
	}
	sb.WriteString(header + strings.Repeat(" ", padding) + help + "\n")
	sb.WriteString(divider + "\n")

	// Search entry input line.
	if m.searchEntry {
		si := m.searchInput
		si.Width = width - 4
		hint := listHelpStyle.Render("  Enter search  Esc cancel")
		sb.WriteString("  " + si.View() + hint + "\n")
		sb.WriteString(divider + "\n")
	}

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
		sb.WriteString(listHelpStyle.Render("  No items.") + "\n")
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
		sb.WriteString(listHelpStyle.Render("  ↑/k up  ↓/j down  enter/→ open  tab toggle view  / search  c create  r refresh  q quit  ? close help") + "\n")
	}

	return sb.String()
}
