# gw-dash

Kanban-style TUI dashboard for monitoring Claude Code agents across [Grove](https://github.com/nicksenap/grove) workspaces.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss). Gruvbox dark theme.

## What it does

`gw dash` shows all active Claude Code sessions in a kanban board:

```
⚡︎ gw dash  agents: 4  >>>2  [?]1  ---1  │  usage: 38% ▓▓▓░░░░░░░ → 3h09m
┌ Active (2) ──────┐┌ Attention (1) ───┐┌ Idle (1) ────────┐┌ Done ───────────┐
│ my-feature  WORK ││ api-refactor WAIT││ hoop-2026  IDLE  ││                 │
│ feat/login       ││ Bash             ││ Edit  49h        ││                 │
│ 82 tools  1h23m  ││ 125 tools  2 err ││ 18 tools         ││                 │
│ ▁▂▃▄▅▆▇█         ││                  ││                  ││                 │
│                  ││                  ││                  ││                 │
│ backend    WORK  ││                  ││                  ││                 │
│ feat/api         ││                  ││                  ││                 │
│ 45 tools  32m    ││                  ││                  ││                 │
└──────────────────┘└──────────────────┘└──────────────────┘└─────────────────┘
q quit  h/l columns  j/k cards  enter jump  y/n approve/deny  r refresh  / search
```

Features:

- **Kanban columns**: Active, Attention (permissions/questions/errors), Idle, Done
- **Detail panel**: workspace info, git branch, model, uptime, tool counts, sparkline activity
- **Zellij integration**: `enter` jumps to the agent's tab, `y`/`n` approves/denies permission requests remotely
- **Claude usage meter**: reads the Usage Tracker cache to show API utilization + reset countdown
- **Live polling**: 500ms refresh, automatic cleanup of dead sessions
- **Search**: `/` to filter agents by name, branch, tool, or status

## Install

Requires [Grove](https://github.com/nicksenap/grove) (Go version) with the plugin system.

```bash
gw plugin install nicksenap/gw-dash
```

Or build from source:

```bash
git clone https://github.com/nicksenap/gw-dash
cd gw-dash
go build -o ~/.grove/plugins/gw-dash .
```

## Usage

```bash
gw dash          # launch the dashboard
gw dash --version
```

### Keybindings

| Key | Action |
|-----|--------|
| `h` / `l` | Move between columns |
| `j` / `k` | Move between cards |
| `enter` | Jump to agent's Zellij tab |
| `y` | Approve permission request |
| `n` | Deny permission request |
| `r` | Refresh |
| `/` | Search / filter |
| `esc` | Clear search |
| `q` | Quit |

## How it works

```
Claude Code hooks ──write──▸ ~/.grove/status/*.json ◂──read── gw-dash
                              (one file per session)
```

Grove's hook handler (`gw _hook`) runs on every Claude Code event (tool use, permission request, notification, etc.) and writes a JSON state file per agent session. `gw-dash` polls these files and renders the kanban board.

The dashboard also reads:
- `~/.grove/state.json` — to resolve workspace names and repos
- `~/.claude/.statusline-usage-cache` — for the API usage meter

### Setup hooks

If not already installed:

```bash
gw dash install    # installs Claude Code hooks
gw dash uninstall  # removes them
```

## Development

```bash
just dev      # build + install to ~/.grove/plugins/ (no push needed)
just run      # run directly without installing
just test     # run tests
just check    # lint + tests
just build    # build binary locally
```

## Architecture

```
main.go                     Entry point
internal/
  grove/
    agent.go                AgentState, StatusSummary, constants
    manager.go              Scan, CleanupStale, BucketAgents, FilterAgents
    usage.go                Claude usage cache reader
    zellij.go               Tab jumping, approve/deny, layout parsing
    grove.go                Grove directory + workspace state reader
  tui/
    model.go                Bubble Tea model, key handling, polling
    render.go               Header, cards, columns, detail panel, status bar
    styles.go               Gruvbox dark palette, shared lipgloss styles
```

## License

MIT
