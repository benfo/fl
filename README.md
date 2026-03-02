# fl — flow

A developer CLI that keeps you in the flow. Manage Jira or Trello tickets, create branches, log notes, and see today's work — all without leaving the terminal.

## Install

```bash
go install github.com/benfo/fl@latest
```

Requires Go 1.23+. The binary is installed as `fl`.

## Setup

### Tracker

```bash
# Jira
fl auth jira

# Trello
fl auth trello
```

### Calendar (optional)

```bash
# Google Calendar
fl auth google

# Outlook / MS 365
fl auth outlook

# iCal feeds — any calendar that exports an .ics URL
fl ical add "Work"     https://calendar.google.com/calendar/ical/.../basic.ics
fl ical add "Personal" webcal://p06-caldav.icloud.com/...
```

Config is written to `~/.fl/config.yaml`. See `config.example.yaml` for all options.

## Commands

### Daily workflow

| Command | Description |
|---|---|
| `fl today` | Today's open items + calendar events at a glance |
| `fl branch [KEY]` | Create a git branch from a tracker item |
| `fl open [KEY]` | Open the current item in your browser |
| `fl note <text> [KEY]` | Add a comment to the current item |
| `fl move [KEY]` | Move an item through its workflow (interactive picker) |

### Creating items

| Command | Description |
|---|---|
| `fl create [summary]` | Create a new item in the current tracker |
| `fl subtask [KEY] <summary>` | Add a subtask to an item |

For Jira, `fl create` shows a picker for project and issue type. `fl subtask` creates a child issue using the project's subtask type.

For Trello, `fl create` shows a picker for board and list. `fl subtask` adds a checklist item to the card (using an existing checklist, or creating one named "Tasks").

### iCal feeds

| Command | Description |
|---|---|
| `fl ical add <name> <url>` | Subscribe to an iCal feed |
| `fl ical list` | List configured iCal feeds |
| `fl ical remove <name>` | Remove an iCal feed |

### Auth & setup

| Command | Description |
|---|---|
| `fl auth jira` | Save Jira credentials to the OS keychain |
| `fl auth trello` | Authenticate with Trello |
| `fl auth google` | Authenticate with Google Calendar via OAuth |
| `fl auth outlook` | Authenticate with Outlook / MS 365 via OAuth |
| `fl init` | Create a `.fl.yaml` config file for the current repo |

For commands that accept an optional `[KEY]`, the item key is inferred from the current git branch name when omitted.

## Per-repo configuration

Run `fl init` inside any git repository to create a `.fl.yaml` file at the repo root. This lets different repos use different trackers or settings. It is safe to commit.

```yaml
# .fl.yaml — repo-level config, overrides ~/.fl/config.yaml

tracker:
  provider: trello   # or: jira

branch:
  template: "{{.Type}}/{{.Key}}-{{.Title}}"
```

See `.fl.example.yaml` for all available options.

## Global configuration

```yaml
# ~/.fl/config.yaml

tracker:
  provider: jira   # or: trello

jira:
  host: https://yourcompany.atlassian.net
  email: you@yourcompany.com
  # Optionally restrict to specific projects:
  # projects:
  #   - PROJ
  #   - DEV

branch:
  # Template variables: .Key  .Title  .Type
  template: "{{.Type}}/{{.Key}}-{{.Title}}"

calendar:
  providers:
    - google
    - outlook
```

API tokens and OAuth tokens are stored in the OS keychain, not in the config file.

## Requirements

- Go 1.23+
- Git
