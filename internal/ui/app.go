package ui

import (
	"github.com/benfo/fl/internal/tracker"
	tea "github.com/charmbracelet/bubbletea"
)

// pushScreenMsg pushes a new screen onto the stack.
type pushScreenMsg struct{ screen tea.Model }

// popScreenMsg pops the top screen off the stack.
type popScreenMsg struct{}

// itemUpdatedMsg signals that an item was changed and the list should refresh.
type itemUpdatedMsg struct{ item *tracker.Item }

// App is the root bubbletea model. It owns a screen stack and routes messages
// to the active (top) screen, handling push/pop itself.
type App struct {
	stack  []tea.Model
	width  int
	height int
}

// NewApp creates a new App with root as the first screen.
func NewApp(root tea.Model) *App {
	return &App{stack: []tea.Model{root}}
}

func (a App) Init() tea.Cmd {
	if len(a.stack) == 0 {
		return nil
	}
	return a.stack[0].Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		// Forward window size to all screens so they can pre-size themselves.
		var cmds []tea.Cmd
		newStack := make([]tea.Model, len(a.stack))
		for i, screen := range a.stack {
			newScreen, cmd := screen.Update(msg)
			newStack[i] = newScreen
			cmds = append(cmds, cmd)
		}
		a.stack = newStack
		return a, tea.Batch(cmds...)

	case pushScreenMsg:
		newScreen := msg.screen
		// Send current dimensions to the new screen immediately.
		newScreen, sizeCmd := newScreen.Update(tea.WindowSizeMsg{Width: a.width, Height: a.height})
		initCmd := newScreen.Init()
		a.stack = append(a.stack, newScreen)
		return a, tea.Batch(initCmd, sizeCmd)

	case popScreenMsg:
		if len(a.stack) > 1 {
			a.stack = a.stack[:len(a.stack)-1]
		}
		return a, nil

	case itemUpdatedMsg:
		// Broadcast to all screens (list refreshes branch indicators, etc.).
		var cmds []tea.Cmd
		newStack := make([]tea.Model, len(a.stack))
		for i, screen := range a.stack {
			newScreen, cmd := screen.Update(msg)
			newStack[i] = newScreen
			cmds = append(cmds, cmd)
		}
		a.stack = newStack
		return a, tea.Batch(cmds...)
	}

	if len(a.stack) == 0 {
		return a, nil
	}
	top := a.stack[len(a.stack)-1]
	newTop, cmd := top.Update(msg)
	a.stack[len(a.stack)-1] = newTop
	return a, cmd
}

func (a App) View() string {
	if len(a.stack) == 0 {
		return ""
	}
	return a.stack[len(a.stack)-1].View()
}
