package tui

type PanelID int

const (
	PanelLeft   PanelID = 0
	PanelCenter PanelID = 1
	PanelRight  PanelID = 2
	NumPanels           = 3
)

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

// Layout — terminal-native, no borders, thin separators.
type Layout struct {
	Width, Height  int
	ToolbarHeight  int
	StatusHeight   int
	PanelHeight    int
	LeftWidth      int
	CenterWidth    int
	RightWidth     int
	SeparatorWidth int
}

var ratios = [3]float64{0.22, 0.48, 0.30}
var minWidths = [3]int{28, 50, 30}

func CalculateLayout(width, height int) Layout {
	l := Layout{
		Width:          width,
		Height:         height,
		ToolbarHeight:  1,
		StatusHeight:   1,
		SeparatorWidth: 1,
	}

	l.PanelHeight = height - l.ToolbarHeight - l.StatusHeight
	if l.PanelHeight < 3 {
		l.PanelHeight = 3
	}

	// Available width after accounting for 2 separators
	avail := width - 2*l.SeparatorWidth
	if avail < 60 {
		avail = 60
	}

	totalRatio := ratios[0] + ratios[1] + ratios[2]
	for i := range ratios {
		ratios[i] /= totalRatio
	}

	raw := [3]int{}
	for i, r := range ratios {
		raw[i] = int(float64(avail) * r)
	}

	// Enforce minimums
	for i, min := range minWidths {
		if raw[i] < min {
			short := min - raw[i]
			raw[i] = min
			for j := range raw {
				if j != i && raw[j] > minWidths[j] {
					raw[j] -= short
					if raw[j] < minWidths[j] {
						raw[j] = minWidths[j]
					}
					break
				}
			}
		}
	}

	l.LeftWidth = raw[0]
	l.CenterWidth = raw[1]
	l.RightWidth = raw[2]

	return l
}
