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
		scoreCards(summary, scaleNames),
		ForestPlot(summary, scaleNames),
		html.Div(
			html.Style("margin-top:2rem"),
			html.A(html.Href("/"), html.Role("button"), g.Text("Start New Assessment")),
		),
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
	mean := ss.RbMean
	std := ss.RbStd
	if mean == 0 && std == 0 {
		mean = ss.Mean
		std = ss.Std
	}

	return html.Div(
		html.Class("score-card"),
		html.H3(g.Text(displayName)),
		html.Div(
			html.Class("score-stats"),
			statBlock("Mean", fmt.Sprintf("%.2f", mean)),
			statBlock("Std", fmt.Sprintf("%.2f", std)),
		),
	)
}

func statBlock(label, value string) g.Node {
	return html.Div(
		html.Div(html.Class("score-stat-label"), g.Text(label)),
		html.Div(html.Class("score-stat-value"), g.Text(value)),
	)
}

// ForestPlot renders an inline SVG forest plot showing credible intervals for each scale.
func ForestPlot(summary *handlers.Summary, scaleNames map[string]string) g.Node {
	keys := sortedScaleKeys(summary.Scores)
	if len(keys) == 0 {
		return g.Group(nil)
	}

	// SVG dimensions
	leftMargin := 160.0
	rightMargin := 30.0
	topMargin := 20.0
	rowHeight := 40.0
	svgWidth := 600.0
	plotWidth := svgWidth - leftMargin - rightMargin
	svgHeight := topMargin + float64(len(keys))*rowHeight + 30

	// Find data range for x-axis
	xMin, xMax := math.MaxFloat64, -math.MaxFloat64
	for _, scale := range keys {
		ss := summary.Scores[scale]
		deciles := ss.RbDeciles
		if len(deciles) == 0 {
			deciles = ss.Deciles
		}
		if len(deciles) >= 9 {
			if deciles[0] < xMin {
				xMin = deciles[0]
			}
			if deciles[8] > xMax {
				xMax = deciles[8]
			}
		}
	}
	// Include 0 in range and add padding
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

	var svgContent strings.Builder

	// Zero line
	zeroX := xScale(0)
	svgContent.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#888" stroke-dasharray="4,4" />`,
		zeroX, topMargin, zeroX, topMargin+float64(len(keys))*rowHeight))

	// Per-scale rows
	for i, scale := range keys {
		ss := summary.Scores[scale]
		displayName := scaleNames[scale]
		if displayName == "" {
			displayName = scale
		}

		y := topMargin + float64(i)*rowHeight + rowHeight/2

		// Label
		svgContent.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" text-anchor="end" font-size="13" fill="currentColor">%s</text>`,
			leftMargin-10, y+4, displayName))

		deciles := ss.RbDeciles
		mean := ss.RbMean
		if len(deciles) == 0 {
			deciles = ss.Deciles
			mean = ss.Mean
		}

		if len(deciles) >= 9 {
			// 80% credible interval: decile[0] (10th) to decile[8] (90th)
			x1 := xScale(deciles[0])
			x2 := xScale(deciles[8])
			svgContent.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="var(--pico-primary)" stroke-width="2" />`,
				x1, y, x2, y))
			// End caps
			svgContent.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="var(--pico-primary)" stroke-width="2" />`,
				x1, y-6, x1, y+6))
			svgContent.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="var(--pico-primary)" stroke-width="2" />`,
				x2, y-6, x2, y+6))
		}

		// Diamond at mean
		mx := xScale(mean)
		d := 5.0
		svgContent.WriteString(fmt.Sprintf(`<polygon points="%.1f,%.1f %.1f,%.1f %.1f,%.1f %.1f,%.1f" fill="var(--pico-primary)" />`,
			mx, y-d, mx+d, y, mx, y+d, mx-d, y))
	}

	// X-axis
	axisY := topMargin + float64(len(keys))*rowHeight + 5
	svgContent.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="currentColor" />`,
		leftMargin, axisY, svgWidth-rightMargin, axisY))

	// Axis ticks
	nTicks := 5
	for i := 0; i <= nTicks; i++ {
		v := xMin + (xMax-xMin)*float64(i)/float64(nTicks)
		x := xScale(v)
		svgContent.WriteString(fmt.Sprintf(`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="currentColor" />`,
			x, axisY, x, axisY+5))
		svgContent.WriteString(fmt.Sprintf(`<text x="%.1f" y="%.1f" text-anchor="middle" font-size="11" fill="currentColor">%.1f</text>`,
			x, axisY+17, v))
	}

	return html.Div(
		html.Class("forest-plot"),
		html.Style("margin:1.5rem 0"),
		g.Raw(fmt.Sprintf(`<svg viewBox="0 0 %.0f %.0f" xmlns="http://www.w3.org/2000/svg">%s</svg>`,
			svgWidth, svgHeight, svgContent.String())),
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
