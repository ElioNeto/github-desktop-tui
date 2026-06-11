package tui

// PanelID identifies which panel has focus.
type PanelID int

const (
	PanelLeft   PanelID = 0
	PanelCenter PanelID = 1
	PanelRight  PanelID = 2
	NumPanels           = 3
)

// ViewID identifies which view is active within a panel.
type ViewID int

const (
	ViewRepositories ViewID = iota
	ViewProviders
	ViewFavorites
	ViewCommitLog
	ViewBranchList
	ViewDiffViewer
	ViewSearch
	ViewDetails
	ViewFileTree
	ViewPreview
	ViewHelp
)

// Layout manages the full terminal dimensions with toolbar, panels, and status bar.
// Desktop-app style: toolbar on top, 3 panels in middle, status bar on bottom.
type Layout struct {
	Width, Height int

	ToolbarHeight int
	StatusHeight  int
	PanelHeight   int

	LeftWidth   int
	CenterWidth int
	RightWidth  int

	LeftMin   int
	CenterMin int
	RightMin  int

	HasBorder bool // whether panels use NormalBorder (1 char border)
}

// Default ratios
var LayoutRatios = [3]float64{0.22, 0.48, 0.30}
var MinWidths = [3]int{30, 52, 32}

// CalculateLayout computes dimensions for the entire screen.
// Layout:
//   Toolbar (1 line)
//   Panels area (toolbar border + panel content)
//   Status bar (1 line)
func CalculateLayout(width, height int) Layout {
	l := Layout{
		Width:         width,
		Height:        height,
		ToolbarHeight: 1,
		StatusHeight:  1,
		HasBorder:     true,
		LeftMin:       MinWidths[0],
		CenterMin:     MinWidths[1],
		RightMin:      MinWidths[2],
	}

	// Panel area = height - toolbar - status
	panelArea := height - l.ToolbarHeight - l.StatusHeight
	if panelArea < 5 {
		panelArea = 5
	}
	l.PanelHeight = panelArea

	// When using borders, each bordered panel takes 1 extra char per side:
	// top border, bottom border, left border, right border.
	// For 3 panels side-by-side: total borders = 4 (2 panel inner + 2 outer) = 2 chars extra per row
	// Actually we render panels with a border function. The border adds 1 char
	// padding. So the content area is reduced by 2 (left + right border) per panel.
	// But we handle this by adjusting content widths.

	// Proportional widths
	totalRatio := 0.0
	for _, r := range LayoutRatios {
		totalRatio += r
	}

	rawWidths := [3]int{}
	for i, ratio := range LayoutRatios {
		rawWidths[i] = int(float64(width) * ratio / totalRatio)
	}

	// Enforce minimums
	for i, min := range MinWidths {
		if rawWidths[i] < min {
			shortfall := min - rawWidths[i]
			rawWidths[i] = min
			for j := range rawWidths {
				if j != i && rawWidths[j] > MinWidths[j] {
					rawWidths[j] -= shortfall
					if rawWidths[j] < MinWidths[j] {
						rawWidths[j] = MinWidths[j]
					}
					break
				}
			}
		}
	}

	l.LeftWidth = rawWidths[0]
	l.CenterWidth = rawWidths[1]
	l.RightWidth = rawWidths[2]

	return l
}

// ContentWidth returns the usable width inside a bordered panel.
func (l Layout) ContentWidth(panel PanelID) int {
	w := 0
	switch panel {
	case PanelLeft:
		w = l.LeftWidth
	case PanelCenter:
		w = l.CenterWidth
	case PanelRight:
		w = l.RightWidth
	}
	if l.HasBorder {
		w -= 2 // left + right border padding
	}
	if w < 2 {
		w = 2
	}
	return w
}
