package frontend

import (
	"fmt"
	"math"
	"sort"
	"strings"

	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/web/handlers"
)

func ResultsPage(sid string, summary *handlers.Summary, scaleNames map[string]string) g.Node {
	return Page("Results",
		Navbar(),
		html.H1(g.Text("Assessment Results")),
		html.P(
			html.Style("color:var(--pico-muted-color)"),
			g.Textf("Session: %s", summary.Session.SessionId),
		),
		html.P(g.Textf("%d questions answered", len(summary.Session.Responses))),
		legend(),
		ForestPlot(summary, scaleNames),
		scoreCards(summary, scaleNames),
		html.Div(
			html.Style("margin-top:2rem;display:flex;gap:1rem"),
			html.A(
				html.Href(fmt.Sprintf("/ui/assess?sid=%s", sid)),
				html.Role("button"),
				html.Class("secondary"),
				g.Text("Continue Assessment"),
			),
			html.A(html.Href("/"), html.Role("button"), g.Text("Start New Assessment")),
		),
	)
}

func legend() g.Node {
	return html.Div(
		html.Class("results-legend"),
		html.Span(
			html.Class("legend-swatch legend-primary"),
		),
		html.Span(g.Text("Marginalized (Rao-Blackwell)")),
		html.Span(
			html.Class("legend-swatch legend-secondary"),
		),
		html.Span(g.Text("Ignoring Missingness")),
	)
}

func scoreCards(summary *handlers.Summary, scaleNames map[string]string) g.Node {
	keys := sortedScaleKeys(summary.Scores)
	var cards []g.Node
	for _, scale := range keys {
		ss := summary.Scores[scale]
		displayName := scaleNames[scale]
		if displayName == "" {
			displayName = scale
		}
		cards = append(cards, ScoreCard(displayName, ss))
	}
	return g.Group(cards)
}

func ScoreCard(displayName string, ss handlers.ScoreSummary) g.Node {
	hasRb := ss.RbMean != 0 || ss.RbStd != 0 || len(ss.RbDensity) > 0

	var statsNodes []g.Node

	// Marginalized (RB) stats
	if hasRb {
		statsNodes = append(statsNodes,
			html.Div(
				html.Class("score-dist-section"),
				html.H4(
					html.Class("score-dist-label dist-primary"),
					g.Text("Marginalized"),
				),
				html.Div(
					html.Class("score-stats"),
					statBlock("Mean", fmt.Sprintf("%.2f", ss.RbMean)),
					statBlock("Std", fmt.Sprintf("%.2f", ss.RbStd)),
					statBlock("Median", medianFromDeciles(ss.RbDeciles)),
				),
			),
		)
	}

	// Observed stats
	statsNodes = append(statsNodes,
		html.Div(
			html.Class("score-dist-section"),
			html.H4(
				html.Class("score-dist-label dist-secondary"),
				g.Text("Ignoring Missingness"),
			),
			html.Div(
				html.Class("score-stats"),
				statBlock("Mean", fmt.Sprintf("%.2f", ss.Mean)),
				statBlock("Std", fmt.Sprintf("%.2f", ss.Std)),
				statBlock("Median", medianFromDeciles(ss.Deciles)),
			),
		),
	)

	children := []g.Node{
		html.H3(g.Text(displayName)),
	}
	children = append(children, statsNodes...)
	children = append(children, DensityPlot(ss))

	return html.Div(
		append([]g.Node{html.Class("score-card")}, children...)...,
	)
}

func medianFromDeciles(deciles []float64) string {
	if len(deciles) >= 5 {
		return fmt.Sprintf("%.2f", deciles[4])
	}
	return "—"
}

func statBlock(label, value string) g.Node {
	return html.Div(
		html.Div(html.Class("score-stat-label"), g.Text(label)),
		html.Div(html.Class("score-stat-value"), g.Text(value)),
	)
}

// DensityPlot renders an inline SVG showing overlaid posterior density traces.
func DensityPlot(ss handlers.ScoreSummary) g.Node {
	if len(ss.Density) == 0 || len(ss.Grid) == 0 {
		return g.Group(nil)
	}

	svgWidth := 500.0
	svgHeight := 160.0
	leftMargin := 10.0
	rightMargin := 10.0
	topMargin := 10.0
	bottomMargin := 25.0
	plotWidth := svgWidth - leftMargin - rightMargin
	plotHeight := svgHeight - topMargin - bottomMargin

	grid := ss.Grid
	density := ss.Density
	rbDensity := ss.RbDensity

	// Find y-range across both densities
	yMax := 0.0
	for _, d := range density {
		if d > yMax {
			yMax = d
		}
	}
	for _, d := range rbDensity {
		if d > yMax {
			yMax = d
		}
	}
	if yMax == 0 {
		yMax = 1
	}

	// X range from grid
	xMin := grid[0]
	xMax := grid[len(grid)-1]

	xScale := func(v float64) float64 {
		return leftMargin + (v-xMin)/(xMax-xMin)*plotWidth
	}
	yScale := func(v float64) float64 {
		return topMargin + plotHeight - (v/yMax)*plotHeight
	}

	var svg strings.Builder

	// Draw observed density (secondary color, behind)
	if len(density) > 0 {
		svg.WriteString(densityPath(grid, density, xScale, yScale, topMargin+plotHeight,
			"var(--pico-secondary)", "var(--pico-secondary-focus)"))
		// Dashed mean line
		mx := xScale(ss.Mean)
		svg.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="var(--pico-secondary)" stroke-width="1.5" stroke-dasharray="4,3" />`,
			mx, topMargin, mx, topMargin+plotHeight))
	}

	// Draw RB density (primary color, on top)
	if len(rbDensity) > 0 {
		svg.WriteString(densityPath(grid, rbDensity, xScale, yScale, topMargin+plotHeight,
			"var(--pico-primary)", "var(--pico-primary-focus)"))
		// Dashed mean line
		mx := xScale(ss.RbMean)
		svg.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="var(--pico-primary)" stroke-width="1.5" stroke-dasharray="4,3" />`,
			mx, topMargin, mx, topMargin+plotHeight))
	}

	// X-axis
	axisY := topMargin + plotHeight
	svg.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="currentColor" stroke-width="0.5" />`,
		leftMargin, axisY, svgWidth-rightMargin, axisY))

	// Axis ticks
	nTicks := 5
	for i := 0; i <= nTicks; i++ {
		v := xMin + (xMax-xMin)*float64(i)/float64(nTicks)
		x := xScale(v)
		svg.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="currentColor" stroke-width="0.5" />`,
			x, axisY, x, axisY+4))
		svg.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="10" fill="currentColor">%.1f</text>`,
			x, axisY+15, v))
	}

	// Theta label
	svg.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="10" fill="var(--pico-muted-color)">%s</text>`,
		svgWidth/2, svgHeight-2, "θ"))

	return html.Div(
		html.Class("density-plot"),
		g.Raw(fmt.Sprintf(`<svg viewBox="0 0 %.0f %.0f" xmlns="http://www.w3.org/2000/svg">%s</svg>`,
			svgWidth, svgHeight, svg.String())),
	)
}

func densityPath(grid, density []float64, xScale, yScale func(float64) float64,
	baseline float64, strokeColor, fillColor string) string {

	n := len(grid)
	if n != len(density) {
		if len(density) < n {
			n = len(density)
		}
	}
	if n == 0 {
		return ""
	}

	var sb strings.Builder

	// Build the filled area path
	sb.WriteString(fmt.Sprintf(`<path d="M%.1f,%.1f`, xScale(grid[0]), baseline))
	for i := 0; i < n; i++ {
		sb.WriteString(fmt.Sprintf(" L%.1f,%.1f", xScale(grid[i]), yScale(density[i])))
	}
	sb.WriteString(fmt.Sprintf(" L%.1f,%.1f Z\"", xScale(grid[n-1]), baseline))
	sb.WriteString(fmt.Sprintf(` fill="%s" fill-opacity="0.3" stroke="%s" stroke-width="1.5" />`, fillColor, strokeColor))

	return sb.String()
}

// ForestPlot renders an inline SVG forest plot showing credible intervals
// for each scale, with both observed and Rao-Blackwellized estimates.
func ForestPlot(summary *handlers.Summary, scaleNames map[string]string) g.Node {
	keys := sortedScaleKeys(summary.Scores)
	if len(keys) == 0 {
		return g.Group(nil)
	}

	// Count how many rows we need (2 per scale if RB data exists, otherwise 1)
	type plotRow struct {
		label   string
		deciles []float64
		mean    float64
		color   string
		isRb    bool
	}
	var rows []plotRow
	for _, scale := range keys {
		ss := summary.Scores[scale]
		displayName := scaleNames[scale]
		if displayName == "" {
			displayName = scale
		}
		hasRb := len(ss.RbDeciles) >= 9

		if hasRb {
			rows = append(rows, plotRow{
				label:   displayName + " (RB)",
				deciles: ss.RbDeciles,
				mean:    ss.RbMean,
				color:   "var(--pico-primary)",
				isRb:    true,
			})
		}
		rows = append(rows, plotRow{
			label:   displayName,
			deciles: ss.Deciles,
			mean:    ss.Mean,
			color:   "var(--pico-secondary)",
		})
	}

	// SVG dimensions
	leftMargin := 200.0
	rightMargin := 30.0
	topMargin := 20.0
	rowHeight := 32.0
	svgWidth := 600.0
	plotWidth := svgWidth - leftMargin - rightMargin
	svgHeight := topMargin + float64(len(rows))*rowHeight + 30

	// Find data range for x-axis
	xMin, xMax := math.MaxFloat64, -math.MaxFloat64
	for _, row := range rows {
		if len(row.deciles) >= 9 {
			if row.deciles[0] < xMin {
				xMin = row.deciles[0]
			}
			if row.deciles[8] > xMax {
				xMax = row.deciles[8]
			}
		}
	}
	if xMin > 0 {
		xMin = 0
	}
	if xMax < 0 {
		xMax = 0
	}
	padding := (xMax - xMin) * 0.1
	if padding < 0.5 {
		padding = 0.5
	}
	xMin -= padding
	xMax += padding

	xScale := func(v float64) float64 {
		return leftMargin + (v-xMin)/(xMax-xMin)*plotWidth
	}

	var svg strings.Builder

	// Zero line
	zeroX := xScale(0)
	svg.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#888" stroke-dasharray="4,4" />`,
		zeroX, topMargin, zeroX, topMargin+float64(len(rows))*rowHeight))

	// Separator lines between scale groups
	rowIdx := 0
	for _, scale := range keys {
		ss := summary.Scores[scale]
		nRows := 1
		if len(ss.RbDeciles) >= 9 {
			nRows = 2
		}
		rowIdx += nRows
		// Draw a subtle separator after each scale group (except the last)
		if rowIdx < len(rows) {
			sepY := topMargin + float64(rowIdx)*rowHeight
			svg.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="currentColor" stroke-opacity="0.15" />`,
				leftMargin-5, sepY, svgWidth-rightMargin, sepY))
		}
	}

	// Per-row rendering
	for i, row := range rows {
		y := topMargin + float64(i)*rowHeight + rowHeight/2

		// Label
		fontSize := "12"
		fontWeight := "normal"
		if row.isRb {
			fontWeight = "bold"
		}
		svg.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" text-anchor="end" font-size="%s" font-weight="%s" fill="currentColor">%s</text>`,
			leftMargin-10, y+4, fontSize, fontWeight, row.label))

		if len(row.deciles) >= 9 {
			// 80% credible interval: decile[0] (10th) to decile[8] (90th)
			x1 := xScale(row.deciles[0])
			x2 := xScale(row.deciles[8])
			svg.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="2" />`,
				x1, y, x2, y, row.color))
			// End caps
			svg.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="2" />`,
				x1, y-5, x1, y+5, row.color))
			svg.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="2" />`,
				x2, y-5, x2, y+5, row.color))

			// IQR box: decile[2] (30th) to decile[6] (70th)
			if len(row.deciles) >= 7 {
				bx1 := xScale(row.deciles[2])
				bx2 := xScale(row.deciles[6])
				svg.WriteString(fmt.Sprintf(`<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="%s" fill-opacity="0.2" stroke="%s" stroke-width="1" />`,
					bx1, y-6, bx2-bx1, 12.0, row.color, row.color))
			}
		}

		// Diamond at mean
		mx := xScale(row.mean)
		d := 4.0
		svg.WriteString(fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="%s" />`,
			mx, y-d, mx+d, y, mx, y+d, mx-d, y, row.color))
	}

	// X-axis
	axisY := topMargin + float64(len(rows))*rowHeight + 5
	svg.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="currentColor" />`,
		leftMargin, axisY, svgWidth-rightMargin, axisY))

	// Axis ticks
	nTicks := 5
	for i := 0; i <= nTicks; i++ {
		v := xMin + (xMax-xMin)*float64(i)/float64(nTicks)
		x := xScale(v)
		svg.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="currentColor" />`,
			x, axisY, x, axisY+5))
		svg.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="11" fill="currentColor">%.1f</text>`,
			x, axisY+17, v))
	}

	// Axis label
	svg.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="11" fill="var(--pico-muted-color)">%s</text>`,
		leftMargin+plotWidth/2, svgHeight-2, "θ (latent trait)"))

	return html.Div(
		html.Class("forest-plot"),
		g.Raw(fmt.Sprintf(`<svg viewBox="0 0 %.0f %.0f" xmlns="http://www.w3.org/2000/svg">%s</svg>`,
			svgWidth, svgHeight, svg.String())),
	)
}

func sortedScaleKeys(scores map[string]handlers.ScoreSummary) []string {
	keys := make([]string, 0, len(scores))
	for k := range scores {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
