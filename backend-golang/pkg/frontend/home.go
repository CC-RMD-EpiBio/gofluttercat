package frontend

import (
	"fmt"
	"sort"

	conf "github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/config"
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

func HomePage(instruments map[string]*handlers.InstrumentRegistry, metas map[string]AssessmentMetaView, catCfg conf.CatConfig) g.Node {
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
			catSettings(catCfg),
			html.Button(
				html.Type("submit"),
				g.Text("Start Assessment"),
			),
		),
	)
}

func catSettings(catCfg conf.CatConfig) g.Node {
	return html.Details(
		html.Class("cat-settings"),
		html.Summary(g.Text("CAT Settings")),
		html.Div(
			html.Class("cat-settings-grid"),
			html.Label(
				g.Attr("for", "stopping_std"),
				g.Text("Stopping threshold (posterior SD)"),
			),
			html.Input(
				html.Type("number"),
				html.ID("stopping_std"),
				html.Name("stopping_std"),
				html.Value(fmt.Sprintf("%.2f", catCfg.StoppingStd)),
				g.Attr("step", "0.01"),
				g.Attr("min", "0.01"),
				g.Attr("max", "5"),
			),
			html.Label(
				g.Attr("for", "stopping_num_items"),
				g.Text("Max items per scale (0 = unlimited)"),
			),
			html.Input(
				html.Type("number"),
				html.ID("stopping_num_items"),
				html.Name("stopping_num_items"),
				html.Value(fmt.Sprintf("%d", catCfg.StoppingNumItems)),
				g.Attr("step", "1"),
				g.Attr("min", "0"),
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
