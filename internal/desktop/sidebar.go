package desktop

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// NewSidebar creates the left panel with branch list and repos.
func NewSidebar(state *AppState) fyne.CanvasObject {
	list := widget.NewList(
		func() int {
			state.mu.RLock()
			n := len(state.branches) + len(state.repos) + 3 // + headers
			state.mu.RUnlock()
			return n
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("Loading...")
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			label := obj.(*widget.Label)
			state.mu.RLock()
			defer state.mu.RUnlock()

			bCount := len(state.branches)
			rCount := len(state.repos)
			_ = rCount

			switch {
			case id == 0:
				label.SetText("▼  Local")
			case id == 1+bCount:
				label.SetText("▼  Remote")
			case id == 2+bCount+rCount:
				label.SetText("▼  Repositories")
			case id > 0 && id <= bCount:
				br := state.branches[id-1]
				if br.IsRemote {
					label.SetText("")
					return
				}
				if br.IsActive {
					txt := "●  " + br.Name
					if br.Ahead > 0 || br.Behind > 0 {
						txt += "  ↑" + itoa(br.Ahead) + "↓" + itoa(br.Behind)
					}
					label.SetText(txt)
				} else {
					label.SetText("   " + br.Name)
				}
			case id > bCount+1 && id <= bCount+1+rCount:
				r := state.repos[id-bCount-2]
				label.SetText("   " + r.Name)
			}
		},
	)

	scroll := container.NewScroll(list)
	scroll.SetMinSize(fyne.NewSize(220, 200))

	// Refresh on data load
	oldRefresh := state.onRefresh
	state.onRefresh = func() {
		fyne.Do(func() {
			list.Refresh()
		})
		if oldRefresh != nil {
			oldRefresh()
		}
	}

	return widget.NewCard("", "", scroll)
}

// itoa converts int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}



