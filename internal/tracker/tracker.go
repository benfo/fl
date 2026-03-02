package tracker

import (
	"fmt"
	"regexp"
)

// ErrIsSubtask is returned by AddSubtask when the target item is itself a
// subtask and therefore cannot have children. It carries the parent's key and
// summary so the caller can offer to redirect the operation.
type ErrIsSubtask struct {
	Key           string // the subtask the user originally targeted
	ParentKey     string
	ParentSummary string
}

func (e *ErrIsSubtask) Error() string {
	return fmt.Sprintf("%s is a subtask; subtasks cannot have children", e.Key)
}

// Item is a generic work item (Jira issue, Trello card, etc.).
type Item struct {
	Key         string
	Summary     string
	Status      string
	Type        string
	URL         string // web URL to open the item; may be empty
	Description string // plain-text description; may be empty
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
	// CreateItem creates a new item at dest with the given summary and optional
	// description. Pass an empty string for description to omit it.
	CreateItem(destID, summary, description string) (*Item, error)
	// AssignToMe assigns the given item to the authenticated user.
	AssignToMe(key string) error
	// AddSubtask adds a subtask to the given item with an optional description.
	// For Jira this creates a child issue; for Trello it adds a checklist item
	// (description is ignored for checklist items).
	// Returns the created item. When the provider uses checklist items rather
	// than independent issues (Trello), item.Key equals parentKey.
	AddSubtask(parentKey, summary, description string) (*Item, error)
	// UpdateItem updates the summary and/or description of an item.
	// Pass empty string for fields that should not be changed.
	UpdateItem(key, summary, description string) error
	// GetSubtasks returns the subtasks (child issues / checklist items) of an item.
	GetSubtasks(parentKey string) ([]*Item, error)
	// UnassignedItems returns open items not assigned to any user.
	UnassignedItems() ([]*Item, error)
	// SearchItems returns items matching the given query string.
	SearchItems(query string) ([]*Item, error)
	// KeyPattern returns a regex for extracting item keys from git branch names.
	KeyPattern() *regexp.Regexp
}
