package desktop

import (
	"image"
	"image/color"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// NewCenterPanel creates the commit graph + list in the center.
func NewCenterPanel(state *AppState) fyne.CanvasObject {
	// Graph canvas for rendering commit topology
	graph := canvas.NewRaster(func(w, h int) image.Image {
		// Render commit graph
		img := image.NewRGBA(image.Rect(0, 0, w, h))
		bgColor := color.RGBA{13, 17, 23, 255} // match theme bg

		// Fill background
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				img.Set(x, y, bgColor)
			}
		}

		state.mu.RLock()
		rows := state.graphRows
		selIdx := state.selectedCommit
		state.mu.RUnlock()

		if len(rows) == 0 {
			return img
		}

		// Calculate layout
		rowHeight := 22
		graphWidth := 40
		commitIdx := 0

		for i, row := range rows {
			y := i * rowHeight
			if y >= h {
				break
			}

			if row.IsCommit {
				// Determine dot color based on column
				dotColor := branchColors[commitIdx%len(branchColors)]

				// Draw connecting lines first (behind dots)
				if i > 0 && len(row.Graph) > 0 {
					prevGraph := ""
					if i-1 < len(rows) {
						prevGraph = rows[i-1].Graph
					}
					_ = prevGraph

					// Draw vertical lines for each column
					for col := 0; col < len(row.Graph) && col < graphWidth; col++ {
						ch := ' '
						if col < len(row.Graph) {
							ch = rune(row.Graph[col])
						}
						g := ch
						if g == '|' || g == '/' || g == '\\' {
							colColor := branchColors[(commitIdx-1+len(branchColors))%len(branchColors)]
							if g == '/' {
								// draw diagonal
								for d := 0; d < 4 && y-d >= 0 && col+d < graphWidth; d++ {
									img.Set(col+d, y-d, colColor)
								}
							} else if g == '\\' {
								for d := 0; d < 4 && y-d >= 0 && col-d >= 0; d++ {
									img.Set(col-d, y-d, colColor)
								}
							} else {
								// vertical line
								for dy := 0; dy < rowHeight && y-dy >= 0; dy++ {
									if col < graphWidth {
										img.Set(col, y-dy, colColor)
									}
								}
							}
						}
					}
				}

				// Draw commit dot
				for dx := -2; dx <= 2; dx++ {
					for dy := -2; dy <= 2; dy++ {
						if dx*dx+dy*dy <= 4 {
							img.Set(5+dx, y+dy, dotColor)
						}
					}
				}

				// Highlight selected
				if commitIdx == selIdx {
					// Draw selection background
					for sx := 0; sx < w; sx++ {
						alpha := uint8(30)
						r, gr, b, _ := bgColor.RGBA()
						_ = r
						_ = gr
						_ = b
						// Light highlight
						img.Set(sx, y, color.RGBA{48, 54, 61, alpha})
					}
				}

				commitIdx++
			}

			// Draw graph lines (| / \)
			_ = rowHeight
		}

		return img
	})

	// Commit list
	commitList := widget.NewList(
		func() int {
			state.mu.RLock()
			n := len(state.commits)
			state.mu.RUnlock()
			return n
		},
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel("     "),
				widget.NewLabel(""),
			)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
			state.mu.RLock()
			if id >= len(state.commits) {
				state.mu.RUnlock()
				return
			}
			c := state.commits[id]
			state.mu.RUnlock()

			box := obj.(*fyne.Container)
			hashLabel := box.Objects[0].(*widget.Label)
			msgLabel := box.Objects[1].(*widget.Label)

			hashLabel.SetText("  " + c.ShortHash + "  ")
			msg := c.MessageHead
			if len(msg) > 60 {
				msg = msg[:60] + "…"
			}
			msgLabel.SetText(msg + "  —  " + c.Author + "  " + c.Timestamp.Format("02 Jan 15:04"))
		},
	)

	// Combine graph + list
	split := container.NewHSplit(
		container.NewScroll(graph),
		container.NewScroll(commitList),
	)
	split.SetOffset(0.15) // 15% graph, 85% list

	// Branch header
	header := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	go func() {
		state.mu.RLock()
		for _, br := range state.branches {
			if br.IsActive {
				header.SetText("▼  " + br.Name)
				break
			}
		}
		state.mu.RUnlock()
	}()

	// Refresh
	oldRefresh := state.onRefresh
	state.onRefresh = func() {
		state.mu.RLock()
		branchName := ""
		for _, br := range state.branches {
			if br.IsActive {
				branchName = br.Name
				break
			}
		}
		state.mu.RUnlock()
		fyne.Do(func() {
			graph.Refresh()
			commitList.Refresh()
			if branchName != "" {
				header.SetText("▼  " + branchName)
			}
		})
		if oldRefresh != nil {
			oldRefresh()
		}
	}

	return container.NewBorder(header, nil, nil, nil, split)
}

// Ensure strings is used
var _ = strings.ToUpper
