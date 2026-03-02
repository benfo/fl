package ui

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/benfo/fl/internal/browser"
	"github.com/benfo/fl/internal/git"
	"github.com/benfo/fl/internal/tracker"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── modes ────────────────────────────────────────────────────────────────────

type detailMode int

const (
	modeNormal        detailMode = iota
	modeEditTitle                // inline textinput replaces summary
	modeEditDesc                 // textarea panel below item info
	modeAddSubtask               // textinput for new subtask summary
	modeTransitions              // overlay picker for status transitions
	modeConfirmBranch            // y/N confirmation when working tree is dirty
)

// ── styles ───────────────────────────────────────────────────────────────────

var (
	detailHeaderStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	detailKeyStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	detailStatusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	detailDivStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	detailLabelStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	detailDescStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	detailHelpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	detailErrStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	detailOkStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	detailSubStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	detailSubSelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	detailOverStyle   = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("99")).
				Padding(0, 1)
	detailPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(lipgloss.Color("99")).
				Padding(0, 1)
)

// ── message types ─────────────────────────────────────────────────────────────

type fullItemLoadedMsg struct {
	item *tracker.Item
	err  error
}

type subtasksLoadedMsg struct {
	subtasks []*tracker.Item
	err      error
}

type branchContextMsg struct {
	branchName   string
	branchExists bool
	isCurrent    bool
}

type transitionsLoadedMsg struct {
	transitions []*tracker.Transition
	err         error
}

type dirtyCheckMsg struct {
	dirty bool
	err   error
}

type branchActionMsg struct {
	msg string
	err error
}

type saveItemMsg struct {
	err error
}

type assignDoneMsg struct {
	err error
}

// ── screen ────────────────────────────────────────────────────────────────────

type itemDetailScreen struct {
	item          *tracker.Item
	itemLoading   bool // true until GetItem returns with full description
	subtasks      []*tracker.Item
	subtaskCursor int
	transitions   []*tracker.Transition
	transCursor   int
	client        tracker.Client
	mode          detailMode
	textarea      textarea.Model
	textinput     textinput.Model
	subtaskInput  textinput.Model
	branchName    string
	branchExists  bool
	isCurrent     bool
	loading       bool
	statusMsg     string
	statusIsErr   bool
	width         int
	height        int
}

func newItemDetailScreen(item *tracker.Item, client tracker.Client) *itemDetailScreen {
	ta := textarea.New()
	ta.Placeholder = "Enter description…"
	ta.CharLimit = 0

	ti := textinput.New()
	ti.Placeholder = "Summary…"
	ti.CharLimit = 256

	si := textinput.New()
	si.Placeholder = "Subtask summary…"
	si.CharLimit = 256

	return &itemDetailScreen{
		item:         item,
		itemLoading:  true,
		client:       client,
		textarea:     ta,
		textinput:    ti,
		subtaskInput: si,
	}
}

func (m itemDetailScreen) Init() tea.Cmd {
	return tea.Batch(m.loadFullItemCmd(), m.loadSubtasksCmd(), m.loadBranchContextCmd())
}

func (m itemDetailScreen) loadFullItemCmd() tea.Cmd {
	client := m.client
	key := m.item.Key
	return func() tea.Msg {
		item, err := client.GetItem(key)
		return fullItemLoadedMsg{item: item, err: err}
	}
}

func (m itemDetailScreen) loadSubtasksCmd() tea.Cmd {
	client := m.client
	key := m.item.Key
	return func() tea.Msg {
		subtasks, err := client.GetSubtasks(key)
		return subtasksLoadedMsg{subtasks: subtasks, err: err}
	}
}

func (m itemDetailScreen) loadBranchContextCmd() tea.Cmd {
	item := m.item
	return func() tea.Msg {
		branchName, err := git.BranchName(item)
		if err != nil {
			return branchContextMsg{}
		}
		branches, _ := git.LocalBranches()
		_, exists := branches[branchName]
		currentBranch, _ := git.CurrentBranch()
		isCurrent := currentBranch == branchName
		return branchContextMsg{branchName: branchName, branchExists: exists, isCurrent: isCurrent}
	}
}

func (m itemDetailScreen) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width - 6)
		m.textarea.SetHeight(8)
		return m, nil

	case fullItemLoadedMsg:
		m.itemLoading = false
		if msg.err == nil {
			// Preserve fields already displayed from the list-screen snapshot
			// (key, summary, status, type) but overwrite with fresh values and
			// add the description that MyOpenItems doesn't fetch.
			m.item = msg.item
		}
		return m, nil

	case subtasksLoadedMsg:
		if msg.err == nil {
			m.subtasks = msg.subtasks
		}
		return m, nil

	case branchContextMsg:
		m.branchName = msg.branchName
		m.branchExists = msg.branchExists
		m.isCurrent = msg.isCurrent
		return m, nil

	case transitionsLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
			m.statusIsErr = true
			m.mode = modeNormal
		} else {
			m.transitions = msg.transitions
			m.transCursor = 0
			m.mode = modeTransitions
		}
		return m, nil

	case dirtyCheckMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
			m.statusIsErr = true
			return m, nil
		}
		if msg.dirty {
			m.mode = modeConfirmBranch
			return m, nil
		}
		// Clean tree — proceed with branch switch.
		return m, m.doBranchCmd()

	case branchActionMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error: %v", msg.err)
			m.statusIsErr = true
		} else {
			m.statusMsg = msg.msg
			m.statusIsErr = false
		}
		m.mode = modeNormal
		return m, tea.Batch(
			m.loadBranchContextCmd(),
			func() tea.Msg { return itemUpdatedMsg{item: m.item} },
		)

	case saveItemMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Error saving: %v", msg.err)
			m.statusIsErr = true
		} else {
			m.statusMsg = "Saved."
			m.statusIsErr = false
		}
		return m, nil

	case assignDoneMsg:
		if msg.err != nil {
			m.statusMsg = fmt.Sprintf("Assign failed: %v", msg.err)
			m.statusIsErr = true
		} else {
			m.statusMsg = "Assigned to you."
			m.statusIsErr = false
		}
		return m, nil
	}

	// Route to mode-specific handler.
	switch m.mode {
	case modeNormal:
		return m.updateNormal(msg)
	case modeEditTitle:
		return m.updateEditTitle(msg)
	case modeEditDesc:
		return m.updateEditDesc(msg)
	case modeAddSubtask:
		return m.updateAddSubtask(msg)
	case modeTransitions:
		return m.updateTransitions(msg)
	case modeConfirmBranch:
		return m.updateConfirmBranch(msg)
	}
	return m, nil
}

// ── mode handlers ─────────────────────────────────────────────────────────────

func (m itemDetailScreen) updateNormal(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	// Clear status message on any keypress.
	m.statusMsg = ""

	switch km.String() {
	case "up", "k":
		if m.subtaskCursor > 0 {
			m.subtaskCursor--
		}
	case "down", "j":
		if m.subtaskCursor < len(m.subtasks)-1 {
			m.subtaskCursor++
		}
	case "enter", "right":
		if len(m.subtasks) > 0 {
			st := m.subtasks[m.subtaskCursor]
			if st.Key != "" && st.Key != m.item.Key {
				detail := newItemDetailScreen(st, m.client)
				return m, func() tea.Msg { return pushScreenMsg{screen: detail} }
			}
		}
	case "esc", "left":
		return m, func() tea.Msg { return popScreenMsg{} }

	case "m":
		m.loading = true
		client := m.client
		key := m.item.Key
		return m, func() tea.Msg {
			transitions, err := client.GetTransitions(key)
			return transitionsLoadedMsg{transitions: transitions, err: err}
		}

	case "e":
		if m.itemLoading {
			m.statusMsg = "Loading item…"
			m.statusIsErr = false
			return m, nil
		}
		m.textinput.SetValue(m.item.Summary)
		m.mode = modeEditTitle
		return m, m.textinput.Focus()

	case "d":
		if m.itemLoading {
			m.statusMsg = "Loading item…"
			m.statusIsErr = false
			return m, nil
		}
		m.textarea.SetValue(m.item.Description)
		m.mode = modeEditDesc
		return m, m.textarea.Focus()

	case "s":
		m.subtaskInput.SetValue("")
		m.mode = modeAddSubtask
		return m, m.subtaskInput.Focus()

	case "b":
		return m, m.checkDirtyCmd()

	case "a":
		client := m.client
		key := m.item.Key
		return m, func() tea.Msg {
			err := client.AssignToMe(key)
			return assignDoneMsg{err: err}
		}

	case "o":
		itemURL := m.item.URL
		if itemURL == "" {
			u, _ := m.client.ItemURL(m.item.Key)
			itemURL = u
		}
		if itemURL != "" {
			_ = browser.Open(itemURL)
		}

	case "p":
		if err := m.openPR(); err != nil {
			m.statusMsg = fmt.Sprintf("PR error: %v", err)
			m.statusIsErr = true
		}
	}
	return m, nil
}

func (m itemDetailScreen) updateEditTitle(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "enter":
			newSummary := strings.TrimSpace(m.textinput.Value())
			m.mode = modeNormal
			if newSummary == "" || newSummary == m.item.Summary {
				return m, nil
			}
			client := m.client
			key := m.item.Key
			m.item.Summary = newSummary
			return m, func() tea.Msg {
				err := client.UpdateItem(key, newSummary, "")
				return saveItemMsg{err: err}
			}
		case "esc":
			m.mode = modeNormal
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textinput, cmd = m.textinput.Update(msg)
	return m, cmd
}

func (m itemDetailScreen) updateEditDesc(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "ctrl+s":
			newDesc := m.textarea.Value()
			m.mode = modeNormal
			m.item.Description = newDesc
			client := m.client
			key := m.item.Key
			return m, func() tea.Msg {
				err := client.UpdateItem(key, "", newDesc)
				return saveItemMsg{err: err}
			}
		case "esc":
			m.mode = modeNormal
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m itemDetailScreen) updateAddSubtask(msg tea.Msg) (tea.Model, tea.Cmd) {
	if km, ok := msg.(tea.KeyMsg); ok {
		switch km.String() {
		case "enter":
			summary := strings.TrimSpace(m.subtaskInput.Value())
			m.mode = modeNormal
			if summary == "" {
				return m, nil
			}
			client := m.client
			key := m.item.Key
			return m, func() tea.Msg {
				_, err := client.AddSubtask(key, summary, "")
				if err != nil {
					return saveItemMsg{err: err}
				}
				subtasks, _ := client.GetSubtasks(key)
				return subtasksLoadedMsg{subtasks: subtasks}
			}
		case "esc":
			m.mode = modeNormal
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.subtaskInput, cmd = m.subtaskInput.Update(msg)
	return m, cmd
}

func (m itemDetailScreen) updateTransitions(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch km.String() {
	case "up", "k":
		if m.transCursor > 0 {
			m.transCursor--
		}
	case "down", "j":
		if m.transCursor < len(m.transitions)-1 {
			m.transCursor++
		}
	case "enter":
		if len(m.transitions) == 0 {
			m.mode = modeNormal
			return m, nil
		}
		t := m.transitions[m.transCursor]
		m.mode = modeNormal
		m.loading = false
		client := m.client
		key := m.item.Key
		transID := t.ID
		transName := t.Name
		m.item.Status = transName
		return m, func() tea.Msg {
			err := client.DoTransition(key, transID)
			if err != nil {
				return saveItemMsg{err: err}
			}
			return saveItemMsg{}
		}
	case "esc", "q":
		m.mode = modeNormal
		m.loading = false
	}
	return m, nil
}

func (m itemDetailScreen) updateConfirmBranch(msg tea.Msg) (tea.Model, tea.Cmd) {
	km, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch km.String() {
	case "y", "Y":
		m.mode = modeNormal
		return m, m.doBranchCmd()
	case "n", "N", "esc":
		m.mode = modeNormal
		m.statusMsg = "Branch switch cancelled."
	}
	return m, nil
}

// ── branch helpers ────────────────────────────────────────────────────────────

func (m itemDetailScreen) checkDirtyCmd() tea.Cmd {
	return func() tea.Msg {
		dirty, err := git.IsWorkingTreeDirty()
		return dirtyCheckMsg{dirty: dirty, err: err}
	}
}

func (m itemDetailScreen) doBranchCmd() tea.Cmd {
	item := m.item
	branchName := m.branchName
	branchExists := m.branchExists
	isCurrent := m.isCurrent
	return func() tea.Msg {
		name := branchName
		if name == "" {
			var err error
			name, err = git.BranchName(item)
			if err != nil {
				return branchActionMsg{err: err}
			}
		}
		if isCurrent {
			return branchActionMsg{msg: fmt.Sprintf("Already on %s", name)}
		}
		if branchExists {
			if err := git.CheckoutBranch(name); err != nil {
				return branchActionMsg{err: err}
			}
			return branchActionMsg{msg: fmt.Sprintf("✓ Switched to %s", name)}
		}
		if err := git.CreateBranch(name); err != nil {
			return branchActionMsg{err: err}
		}
		return branchActionMsg{msg: fmt.Sprintf("✓ Created and switched to %s", name)}
	}
}

// ── PR URL ────────────────────────────────────────────────────────────────────

func (m itemDetailScreen) openPR() error {
	branchName := m.branchName
	if branchName == "" {
		var err error
		branchName, err = git.BranchName(m.item)
		if err != nil {
			return err
		}
	}
	remoteURL, err := git.RemoteURL("origin")
	if err != nil {
		return err
	}
	host, owner, repo, err := git.ParseRemoteURL(remoteURL)
	if err != nil {
		return err
	}
	base := git.DefaultBaseBranch()
	title := fmt.Sprintf("%s: %s", m.item.Key, m.item.Summary)
	body, _ := m.client.ItemURL(m.item.Key)

	repoBase := fmt.Sprintf("https://%s/%s/%s", host, owner, repo)
	var prURL string
	if strings.Contains(host, "gitlab") {
		q := url.Values{}
		q.Set("merge_request[source_branch]", branchName)
		q.Set("merge_request[target_branch]", base)
		q.Set("merge_request[title]", title)
		if body != "" {
			q.Set("merge_request[description]", body)
		}
		prURL = repoBase + "/-/merge_requests/new?" + q.Encode()
	} else {
		q := url.Values{}
		q.Set("quick_pull", "1")
		q.Set("title", title)
		if body != "" {
			q.Set("body", body)
		}
		prURL = fmt.Sprintf("%s/compare/%s...%s?%s", repoBase, base, branchName, q.Encode())
	}
	return browser.Open(prURL)
}

// ── view ─────────────────────────────────────────────────────────────────────

func (m itemDetailScreen) View() string {
	width := m.width
	if width == 0 {
		width = 80
	}
	divider := detailDivStyle.Render(strings.Repeat("─", width))

	var sb strings.Builder

	// Header
	backLabel := detailHelpStyle.Render("← ")
	key := detailKeyStyle.Render(m.item.Key)
	status := detailStatusStyle.Render(fmt.Sprintf("  [%s]", m.item.Status))
	issueType := detailHelpStyle.Render(fmt.Sprintf("  %s", m.item.Type))
	sb.WriteString(backLabel + key + status + issueType + "\n")
	sb.WriteString(divider + "\n")

	// Summary (or textinput in edit mode)
	if m.mode == modeEditTitle {
		ti := m.textinput
		ti.Width = width - 4
		sb.WriteString("  " + ti.View() + "\n")
	} else {
		sb.WriteString("  " + detailHeaderStyle.Render(m.item.Summary) + "\n")
	}
	sb.WriteString("\n")

	// Loading indicator for description area.
	if m.itemLoading {
		sb.WriteString("  " + detailHelpStyle.Render("Loading…") + "\n\n")
	}

	// Description edit panel — early return so nothing else renders below it.
	if m.mode == modeEditDesc {
		sb.WriteString(divider + "\n")
		hint := detailHelpStyle.Render("Ctrl+S save  Esc cancel")
		sb.WriteString("  " + detailLabelStyle.Render("Edit description") + "  " + hint + "\n")
		ta := m.textarea
		sb.WriteString(detailPanelStyle.Width(width-4).Render(ta.View()) + "\n")
		return sb.String()
	}

	// Description display
	if m.item.Description != "" {
		sb.WriteString("  " + detailLabelStyle.Render("Description") + "\n")
		for _, line := range strings.Split(m.item.Description, "\n") {
			sb.WriteString("  " + detailDescStyle.Render(line) + "\n")
		}
		sb.WriteString("\n")
	}

	// Subtasks
	if len(m.subtasks) > 0 {
		sb.WriteString("  " + detailLabelStyle.Render(fmt.Sprintf("Subtasks (%d)", len(m.subtasks))) + "\n")
		for i, st := range m.subtasks {
			cursor := "  "
			style := detailSubStyle
			if i == m.subtaskCursor {
				cursor = "▶ "
				style = detailSubSelStyle
			}
			circle := "○"
			if strings.EqualFold(st.Status, "complete") || strings.EqualFold(st.Status, "done") {
				circle = "●"
			}
			keyPart := ""
			if st.Key != "" {
				keyPart = st.Key + "  "
			}
			sb.WriteString("  " + cursor + style.Render(circle+"  "+keyPart+truncate(st.Summary, width-20)) + "\n")
		}
		sb.WriteString("\n")
	}

	// Add-subtask input panel — early return.
	if m.mode == modeAddSubtask {
		sb.WriteString(divider + "\n")
		sb.WriteString("  " + detailLabelStyle.Render("New subtask") + "  " + detailHelpStyle.Render("Enter confirm  Esc cancel") + "\n")
		si := m.subtaskInput
		si.Width = width - 4
		sb.WriteString("  " + si.View() + "\n")
		return sb.String()
	}

	// Transition overlay — renders after the rest of the content.
	if m.mode == modeTransitions {
		sb.WriteString(m.renderTransitionOverlay(width))
		return sb.String()
	}

	// Dirty-tree confirmation.
	if m.mode == modeConfirmBranch {
		sb.WriteString(divider + "\n")
		sb.WriteString("  " + detailErrStyle.Render("Uncommitted changes detected. Switch branch anyway? [y/N]") + "\n")
		return sb.String()
	}

	// Status message
	if m.statusMsg != "" {
		if m.statusIsErr {
			sb.WriteString("  " + detailErrStyle.Render(m.statusMsg) + "\n\n")
		} else {
			sb.WriteString("  " + detailOkStyle.Render(m.statusMsg) + "\n\n")
		}
	}

	// Footer
	sb.WriteString(divider + "\n")
	if m.branchName != "" {
		sb.WriteString("  " + detailHelpStyle.Render(fmt.Sprintf("branch: %s", m.branchName)) + "\n")
	}
	sb.WriteString("  " + detailHelpStyle.Render("m move  e title  d desc  s subtask  b branch  p PR  a assign  o open  ← back") + "\n")

	return sb.String()
}

func (m itemDetailScreen) renderTransitionOverlay(width int) string {
	var inner strings.Builder
	inner.WriteString(detailHeaderStyle.Render(fmt.Sprintf("Move %s to…", m.item.Key)) + "\n\n")
	for i, t := range m.transitions {
		cursor := "  "
		style := detailHelpStyle
		if i == m.transCursor {
			cursor = "▶ "
			style = detailSubSelStyle
		}
		inner.WriteString(cursor + style.Render(t.Name) + "\n")
	}
	inner.WriteString("\n" + detailHelpStyle.Render("↑/↓ navigate  enter select  esc cancel"))

	overlayWidth := 40
	if width > 60 {
		overlayWidth = width / 2
	}
	return detailOverStyle.Width(overlayWidth).Render(inner.String()) + "\n"
}
