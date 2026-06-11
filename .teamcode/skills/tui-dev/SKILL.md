---
name: tui-dev
description: >
  Use when developing TUI (Terminal User Interface) components, views, widgets,
  keybindings, layouts, or terminal rendering. Also use when working with
  Bubble Tea, Ratatui, Ink, tview, or similar TUI frameworks.
---

# TUI Development

This skill covers patterns and conventions for building the terminal UI of
github-desktop-tui.

## TUI Frameworks (detect & use the right one)

| Framework | Language | File Extension | Key Package |
|-----------|----------|----------------|-------------|
| Bubble Tea | Go | `.go` | `github.com/charmbracelet/bubbletea` |
| Ratatui | Rust | `.rs` | `ratatui` |
| Ink | TypeScript | `.tsx` | `ink` |
| tview | Go | `.go` | `github.com/rivo/tview` |

## Component Structure

Each view should follow this pattern:

```
src/tui/views/
  <view-name>/
    view.go          # Main view component (or view.tsx, view.rs)
    model.go          # State model
    update.go         # Update logic / message handling
    view_helpers.go   # Rendering helpers
```

## Keybindings

All keybindings should be registered in `src/tui/keybindings/`:

| Key | Action | Context |
|-----|--------|---------|
| `q` / `Ctrl+C` | Quit | Global |
| `?` | Toggle help | Global |
| `Tab` | Next panel | Global |
| `Enter` | Confirm/select | Contextual |
| `/` | Search/filter | List views |
| `r` | Refresh | All views |
| `d` | Diff view | Commit list |

## Styling

- Use the theme system (`src/tui/theme/`) — never hardcode colors
- Colors are defined in `.teamcode/themes/mytheme.json`
- Support both truecolor and 256-color fallback
- Respect `NO_COLOR` environment variable

## Terminal Size Handling

Always handle resize events:
```go
// Go (Bubble Tea)
type WindowSizeMsg tea.WindowSizeMsg

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        // Recalculate layouts
    }
}
```

```typescript
// React/Ink
const {columns, rows} = useStdoutDimensions();
// Re-render on resize automatically via useStdoutDimensions
```
