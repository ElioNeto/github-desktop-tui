package desktop

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// NewToolbar creates the top toolbar with repo info, branch, and actions.
func NewToolbar(state *AppState, onRefresh func()) fyne.CanvasObject {
	// App icon/title
	title := canvas.NewText("◉  github-desktop-tui", color.RGBA{255, 255, 255, 255})
	title.TextSize = 16

	// Branch label
	branchLabel := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	updateBranch := func() {
		state.mu.RLock()
		b := ""
		if len(state.branches) > 0 {
			for _, br := range state.branches {
				if br.IsActive {
					b = br.Name
					break
				}
			}
		}
		if b == "" {
			b = "main"
		}
		state.mu.RUnlock()
		branchLabel.SetText("⎇  " + b)
	}
	updateBranch()

	// Branch selector (combo)
	branchSelect := widget.NewSelect([]string{}, func(s string) {
		// TODO: checkout selected branch
		state.mu.RLock()
		for _, br := range state.branches {
			if br.Name == s {
				go func() {
					state.gitOps.Checkout(nil, br.Name)
					state.LoadData()
				}()
				break
			}
		}
		state.mu.RUnlock()
	})
	branchSelect.PlaceHolder = "Switch branch..."

	// Update branch list
	go func() {
		state.mu.RLock()
		names := make([]string, 0, len(state.branches))
		for _, br := range state.branches {
			if !br.IsRemote {
				names = append(names, br.Name)
			}
		}
		state.mu.RUnlock()
		branchSelect.Options = names
		branchSelect.Refresh()
		updateBranch()
	}()

	// Action buttons
	refreshBtn := widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), onRefresh)

	pushBtn := widget.NewButtonWithIcon("", theme.MailSendIcon(), func() {
		state.mu.RLock()
		branch := ""
		for _, br := range state.branches {
			if br.IsActive {
				branch = br.Name
				break
			}
		}
		state.mu.RUnlock()
		if branch != "" {
			go func() {
				state.gitOps.Push(nil, "origin", branch, false)
				state.LoadData()
			}()
		}
	})

	pullBtn := widget.NewButtonWithIcon("", theme.DownloadIcon(), func() {
		go func() {
			state.gitOps.Pull(nil, "origin", "")
			state.LoadData()
		}()
	})

	// Search entry
	search := widget.NewEntry()
	search.PlaceHolder = "Search commits..."

	// Commit button (accent)
	commitBtn := widget.NewButton("Commit", func() {
		// TODO: open commit dialog
	})

	// Build toolbar layout
	// Fyne NewBorder(top, bottom, left, right)
	leftBox := container.NewHBox(
		title,
		widget.NewLabel("  "),
		branchLabel,
		widget.NewLabel("  "),
		branchSelect,
	)
	rightBox := container.NewHBox(
		search,
		widget.NewLabel("  "),
		commitBtn,
		pushBtn,
		pullBtn,
		refreshBtn,
	)
	toolbarContent := container.NewBorder(nil, nil, leftBox, rightBox)

	// Orange background bar
	bg := canvas.NewRectangle(color.RGBA{239, 108, 0, 255})
	toolbar := container.NewMax(bg, container.NewPadded(toolbarContent))

	// Wrap in a border-less layout
	return container.NewVBox(
		toolbar,
		canvas.NewLine(color.RGBA{48, 54, 61, 255}),
	)
}


