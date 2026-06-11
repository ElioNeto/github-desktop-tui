package desktop

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// NewDetailsPanel creates the right panel with commit details and changed files.
func NewDetailsPanel(state *AppState) fyne.CanvasObject {
	// Commit details
	hashLabel := widget.NewLabel("")
	authorLabel := widget.NewLabel("")
	dateLabel := widget.NewLabel("")
	msgLabel := widget.NewLabel("")
	msgLabel.Wrapping = fyne.TextWrapWord

	// Files changed list
	fileList := widget.NewList(
		func() int {
			state.mu.RLock()
			n := len(state.changes)
			state.mu.RUnlock()
			return n
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			state.mu.RLock()
			if id >= len(state.changes) {
				state.mu.RUnlock()
				return
			}
			fc := state.changes[id]
			state.mu.RUnlock()

			label := obj.(*widget.Label)
			badge := " "
			switch fc.Status {
			case 0:
				badge = " "
			case 1:
				badge = "A"
			case 2:
				badge = "D"
			case 3:
				badge = "M"
			case 7:
				badge = "?"
			}
			label.SetText(fmt.Sprintf("  %s  %s", badge, fc.Path))
		},
	)

	// Refresh function
	update := func() {
		state.mu.RLock()
		commits := state.commits
		selIdx := state.selectedCommit
		state.mu.RUnlock()

		if selIdx >= 0 && selIdx < len(commits) {
			c := commits[selIdx]
			hashLabel.SetText("Hash:     " + c.Hash[:12])
			authorLabel.SetText("Author:   " + c.Author)
			dateLabel.SetText("Date:     " + c.Timestamp.Format("02 Jan 2006 15:04"))
			msg := c.MessageHead
			if len(msg) > 80 {
				msg = msg[:80] + "…"
			}
			msgLabel.SetText("Message:  " + msg)
		} else {
			hashLabel.SetText("No commit selected")
			authorLabel.SetText("")
			dateLabel.SetText("")
			msgLabel.SetText("")
		}

		fileList.Refresh()
	}

	// Initial update
	update()

	// Hook into state refresh
	oldRefresh := state.onRefresh
	state.onRefresh = func() {
		update()
		if oldRefresh != nil {
			oldRefresh()
		}
	}

	// Layout
	details := container.NewVBox(
		widget.NewLabelWithStyle("Commit", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		hashLabel,
		authorLabel,
		dateLabel,
		msgLabel,
		widget.NewSeparator(),
		widget.NewLabelWithStyle("Files Changed", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
	)

	scroll := container.NewScroll(details)
	scroll.SetMinSize(fyne.NewSize(280, 200))

	return container.NewBorder(
		nil, nil, nil, nil,
		container.NewVSplit(
			scroll,
			container.NewScroll(fileList),
		),
	)
}

// Ensure fmt used
var _ = fmt.Sprintf
