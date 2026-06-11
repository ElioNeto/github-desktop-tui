package tui

// PanelID identifies which panel currently has focus.
type PanelID int

const (
	PanelLeft   PanelID = 0 // Explorer (repos, providers)
	PanelCenter PanelID = 1 // Content (commits, diff, branches)
	PanelRight  PanelID = 2 // Details (file tree, preview)

	NumPanels = 3
)

// ViewID identifies which view is active within a panel.
type ViewID int

const (
	// Left panel views
	ViewRepositories ViewID = iota
	ViewProviders
	ViewFavorites

	// Center panel views
	ViewCommitLog
	ViewBranchList
	ViewDiffViewer
	ViewSearch

	// Right panel views
	ViewDetails
	ViewFileTree
	ViewPreview

	// Overlay views
	ViewHelp
)

// Layout manages the size and position of the three main panels.
type Layout struct {
	Width  int
	Height int

	LeftWidth   int
	CenterWidth int
	RightWidth  int

	StatusBarHeight int
	PanelHeight     int
}

// LayoutRatios defines the default width ratios for the three panels.
var LayoutRatios = [3]float64{0.25, 0.45, 0.30}

// MinWidths defines the minimum width for each panel.
var MinWidths = [3]int{30, 50, 35}

// CalculateLayout computes panel dimensions from total terminal size.
func CalculateLayout(width, height int) Layout {
	l := Layout{
		Width:           width,
		Height:          height,
		StatusBarHeight: 1,
	}

	// Reserve space for status bar
	l.PanelHeight = height - l.StatusBarHeight
	if l.PanelHeight < 1 {
		l.PanelHeight = 1
	}

	// Calculate panel widths based on ratios, respecting minimums
	totalRatio := 0.0
	for _, r := range LayoutRatios {
		totalRatio += r
	}

	// First pass: calculate proportional widths
	rawWidths := [3]int{}
	for i, ratio := range LayoutRatios {
		rawWidths[i] = int(float64(width) * ratio / totalRatio)
	}

	// Second pass: enforce minimum widths
	for i, min := range MinWidths {
		if rawWidths[i] < min {
			// Take from the largest panel
			shortfall := min - rawWidths[i]
			rawWidths[i] = min
			// Find the largest other panel to give space
			largestIdx := -1
			largestVal := 0
			for j := range rawWidths {
				if j != i && rawWidths[j] > MinWidths[j] && rawWidths[j] > largestVal {
					largestIdx = j
					largestVal = rawWidths[j]
				}
			}
			if largestIdx >= 0 {
				rawWidths[largestIdx] -= shortfall
				if rawWidths[largestIdx] < MinWidths[largestIdx] {
					rawWidths[largestIdx] = MinWidths[largestIdx]
				}
			}
		}
	}

	l.LeftWidth = rawWidths[0]
	l.CenterWidth = rawWidths[1]
	l.RightWidth = rawWidths[2]

	return l
}

// PanelBounds returns the x, y, width, height for a given panel.
func (l Layout) PanelBounds(panel PanelID) (x, y, width, height int) {
	y = 0
	height = l.PanelHeight

	switch panel {
	case PanelLeft:
		x = 0
		width = l.LeftWidth
	case PanelCenter:
		x = l.LeftWidth
		width = l.CenterWidth
	case PanelRight:
		x = l.LeftWidth + l.CenterWidth
		width = l.RightWidth
	}

	return x, y, width, height
}
