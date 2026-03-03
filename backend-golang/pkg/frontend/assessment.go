package frontend

import (
	"fmt"
	"sort"
	"strings"

	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
	hx "maragu.dev/gomponents-htmx"

	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/irtcat"
	"github.com/CC-RMD-EpiBio/gofluttercat/backend-golang/pkg/web/handlers"
)

func AssessmentPage(sid string, item *handlers.ItemServed, numResponses int) g.Node {
	return Page("Assessment",
		Navbar(),
		html.Div(
			html.ID("question-area"),
			html.Class("question-area"),
			QuestionCard(sid, item, numResponses),
		),
	)
}

func QuestionCard(sid string, item *handlers.ItemServed, numResponses int) g.Node {
	if item == nil {
		return html.Div(
			html.P(g.Text("Assessment complete. Loading results...")),
			html.Script(g.Raw(fmt.Sprintf(`window.location.href="/ui/results?sid=%s";`, sid))),
		)
	}

	progress := html.P(
		html.Style("color:var(--pico-muted-color);font-size:0.85rem"),
		g.Textf("Question %d", numResponses+1),
	)

	question := html.H2(
		html.Style("font-size:1.25rem"),
		g.Text(item.Question),
	)

	// Separate skip choices from regular choices
	type choiceEntry struct {
		Key   string
		Value irtcat.Choice
	}
	var regular []choiceEntry
	var skipChoice *irtcat.Choice
	for k, c := range item.Choices {
		if isSkipChoice(c) {
			cc := c
			skipChoice = &cc
		} else {
			regular = append(regular, choiceEntry{Key: k, Value: c})
		}
	}
	sort.Slice(regular, func(i, j int) bool { return regular[i].Key < regular[j].Key })

	var choiceButtons []g.Node
	for i, c := range regular {
		num := i + 1
		choiceButtons = append(choiceButtons, html.Button(
			html.Class("choice-btn secondary outline"),
			html.Type("button"),
			hx.Post("/ui/respond"),
			hx.Target("#question-area"),
			hx.Swap("innerHTML"),
			hx.Vals(fmt.Sprintf(`{"sid":"%s","item_name":"%s","value":%d}`, sid, item.Name, c.Value.Value)),
			html.Span(html.Class("choice-number"), g.Textf("%d.", num)),
			g.Textf(" %s", c.Value.Text),
		))
	}

	// Skip button - use the explicit skip choice value if available, otherwise -1
	skipValue := -1
	if skipChoice != nil {
		skipValue = int(skipChoice.Value)
	}
	skipBtn := html.Button(
		html.Class("skip-btn secondary outline"),
		html.Type("button"),
		html.Style("width:100%"),
		hx.Post("/ui/respond"),
		hx.Target("#question-area"),
		hx.Swap("innerHTML"),
		hx.Vals(fmt.Sprintf(`{"sid":"%s","item_name":"%s","value":%d}`, sid, item.Name, skipValue)),
		g.Text("Skip"),
	)

	return g.Group([]g.Node{
		progress,
		question,
		html.Div(html.Class("choice-grid"), g.Group(choiceButtons)),
		skipBtn,
	})
}

func isSkipChoice(c irtcat.Choice) bool {
	return strings.EqualFold(c.Text, "skip")
}
