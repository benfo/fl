package tracker

import "regexp"

// Item is a generic work item (Jira issue, Trello card, etc.).
type Item struct {
	Key     string
	Summary string
	Status  string
	Type    string
}

// Transition is a state the item can move to (Jira status, Trello list).
type Transition struct {
	ID   string
	Name string
}

// CreateDest is a destination where a new item can be created.
// For Jira this is a (project, issue type) pair; for Trello a board list.
type CreateDest struct {
	ID    string // opaque; passed back to CreateItem
	Label string // human-readable, e.g. "PROJ · Task" or "My Board · To Do"
}

// Client is the interface all tracker backends must implement.
type Client interface {
	GetItem(key string) (*Item, error)
	ItemURL(key string) (string, error)
	AddComment(key, text string) error
	GetTransitions(key string) ([]*Transition, error)
	DoTransition(key, transitionID string) error
	MyOpenItems() ([]*Item, error)
	// CreateDests returns the available destinations for item creation.
	CreateDests() ([]*CreateDest, error)
	// CreateItem creates a new item at dest with the given summary and returns it.
	CreateItem(destID, summary string) (*Item, error)
	// AddSubtask adds a subtask to the given item.
	// For Jira this creates a child issue; for Trello it adds a checklist item.
	// Returns the created item. When the provider uses checklist items rather
	// than independent issues (Trello), item.Key equals parentKey.
	AddSubtask(parentKey, summary string) (*Item, error)
	// KeyPattern returns a regex for extracting item keys from git branch names.
	KeyPattern() *regexp.Regexp
}
