# fl — flow

A developer CLI that keeps you in the flow. Manage Jira tickets, create branches, log notes, and see today's work — all without leaving the terminal.

## Install

```bash
git clone https://github.com/benfourie/fl
cd fl
go build -o fl .
# Move the binary somewhere on your PATH, e.g.:
mv fl /usr/local/bin/fl
```

## Setup

```bash
# Jira (required)
fl auth jira

# Google Calendar (optional)
fl auth google

# Outlook / MS 365 (optional)
fl auth outlook

# iCal feeds — any calendar that exports an .ics URL (optional)
fl ical add "Work"     https://calendar.google.com/calendar/ical/.../basic.ics
fl ical add "Personal" webcal://p06-caldav.icloud.com/...
```

Config is written to `~/.fl/config.yaml`. See `config.example.yaml` for all options.

## Commands

| Command | Description |
|---|---|
| `fl today` | Today's Jira tickets + calendar events at a glance |
| `fl branch [KEY]` | Create a git branch from a Jira ticket |
| `fl open [KEY]` | Open the current ticket in your browser |
| `fl note "text"` | Add a comment to the current ticket |
| `fl move [KEY]` | Move a ticket to the next workflow step |
| `fl ical add <name> <url>` | Subscribe to an iCal feed |
| `fl ical list` | List configured iCal feeds |
| `fl ical remove <name>` | Remove an iCal feed |

For commands that accept an optional `[KEY]`, the ticket key is inferred from the current git branch name when omitted.

## Configuration

```yaml
# ~/.fl/config.yaml

jira:
  host: https://yourcompany.atlassian.net
  email: you@yourcompany.com
  # Optionally restrict to specific projects
  # projects:
  #   - PROJ
  #   - DEV

branch:
  # Template variables: .Key  .Title  .Type
  template: "{{.Type}}/{{.Key}}-{{.Title}}"
```

API tokens and OAuth tokens are stored in the OS keychain, not in the config file.

## Requirements

- Go 1.23+
- Git
