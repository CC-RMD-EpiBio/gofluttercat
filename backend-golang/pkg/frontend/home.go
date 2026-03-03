package frontend

import (
	"sort"

	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
	hx "maragu.dev/gomponents-htmx"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/web/handlers"
)

type instrumentView struct {
	ID          string
	Name        string
	Description string
	Scales      map[string]string
}

func HomePage(instruments map[string]*handlers.InstrumentRegistry, metas map[string]AssessmentMetaView) g.Node {
	views := make([]instrumentView, 0, len(instruments))
	for id := range instruments {
		meta, ok := metas[id]
		if !ok {
			continue
		}
		views = append(views, instrumentView{
			ID:          id,
			Name:        meta.Name,
			Description: meta.Description,
			Scales:      meta.Scales,
		})
	}
	sort.Slice(views, func(i, j int) bool { return views[i].ID < views[j].ID })

	firstID := ""
	if len(views) > 0 {
		firstID = views[0].ID
	}

	return Page("Home",
		Navbar(),
		html.H1(g.Text("Computer Adaptive Testing")),
		html.P(g.Text("Select an instrument and begin your assessment.")),
		html.Form(
			html.ID("start-form"),
			hx.Post("/ui/start"),
			hx.Swap("none"),
			instrumentList(views, firstID),
			html.Button(
				html.Type("submit"),
				g.Text("Start Assessment"),
			),
		),
	)
}

func instrumentList(instruments []instrumentView, defaultID string) g.Node {
	var cards []g.Node
	for _, inst := range instruments {
		checked := inst.ID == defaultID
		cards = append(cards, instrumentCard(inst, checked))
	}
	return html.Div(g.Group(cards))
}

func instrumentCard(inst instrumentView, checked bool) g.Node {
	attrs := []g.Node{
		html.Class("instrument-card"),
		html.Label(
			html.Input(
				html.Type("radio"),
				html.Name("instrument"),
				html.Value(inst.ID),
				g.If(checked, html.Checked()),
			),
			html.Strong(g.Text(inst.Name)),
		),
		html.P(
			html.Style("margin:0.25rem 0 0 1.75rem;font-size:0.9rem"),
			g.Text(inst.Description),
		),
		scaleChips(inst.Scales),
	}
	return html.Div(attrs...)
}

func scaleChips(scales map[string]string) g.Node {
	names := make([]string, 0, len(scales))
	for _, displayName := range scales {
		names = append(names, displayName)
	}
	sort.Strings(names)

	var chips []g.Node
	for _, name := range names {
		chips = append(chips, html.Span(html.Class("scale-chip"), g.Text(name)))
	}
	return html.Div(html.Class("scale-chips"), html.Style("margin-left:1.75rem"), g.Group(chips))
}
