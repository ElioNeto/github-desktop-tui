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
// Glint-style: no borders, thin separators, full-width content.
type Layout struct {
	Width  int
	Height int

	LeftWidth   int
	CenterWidth int
	RightWidth  int

	SeparatorWidth int // thin gap between panels (1 char)
	StatusBarLines int  // lines for status + command bar
	PanelHeight    int
}

// LayoutRatios defines the default width ratios for the three panels.
var LayoutRatios = [3]float64{0.22, 0.50, 0.28}

// MinWidths defines the minimum width for each panel.
var MinWidths = [3]int{28, 50, 30}

// CalculateLayout computes panel dimensions from total terminal size.
// Glint-style: panels are separated by a 1-char vertical line.
func CalculateLayout(width, height int) Layout {
	l := Layout{
		Width:          width,
		Height:         height,
		SeparatorWidth: 1, // thin vertical line between panels
		StatusBarLines: 2, // status bar + command bar
	}

	// Reserve space for status + command bars
	l.PanelHeight = height - l.StatusBarLines
	if l.PanelHeight < 1 {
		l.PanelHeight = 1
	}

	// Calculate available width for panels (accounting for separators)
	totalSeparators := (NumPanels - 1) * l.SeparatorWidth
	availWidth := width - totalSeparators
	if availWidth < 60 {
		availWidth = 60
	}

	// Calculate proportional widths
	totalRatio := 0.0
	for _, r := range LayoutRatios {
		totalRatio += r
	}

	rawWidths := [3]int{}
	for i, ratio := range LayoutRatios {
		rawWidths[i] = int(float64(availWidth) * ratio / totalRatio)
	}

	// Enforce minimum widths
	for i, min := range MinWidths {
		if rawWidths[i] < min {
			shortfall := min - rawWidths[i]
			rawWidths[i] = min
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
// x accounts for separators.
func (l Layout) PanelBounds(panel PanelID) (x, y, width, height int) {
	y = 0
	height = l.PanelHeight

	switch panel {
	case PanelLeft:
		x = 0
		width = l.LeftWidth
	case PanelCenter:
		x = l.LeftWidth + l.SeparatorWidth
		width = l.CenterWidth
	case PanelRight:
		x = l.LeftWidth + l.SeparatorWidth + l.CenterWidth + l.SeparatorWidth
		width = l.RightWidth
	}

	return x, y, width, height
}

// SeparatorX returns the x position of the separator before a panel.
func (l Layout) SeparatorX(panel PanelID) int {
	switch panel {
	case PanelCenter:
		return l.LeftWidth
	case PanelRight:
		return l.LeftWidth + l.SeparatorWidth + l.CenterWidth
	default:
		return 0
	}
}
