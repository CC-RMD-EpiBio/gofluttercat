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
			html.Style("display:flex;justify-content:space-between;align-items:center;margin-bottom:0.5rem"),
			html.A(
				html.Href("/"),
				html.Style("font-size:0.85rem"),
				g.Text("Home"),
			),
			html.A(
				html.Href(fmt.Sprintf("/ui/results?sid=%s", sid)),
				html.Style("font-size:0.85rem"),
				g.Text("View Results"),
			),
		),
		html.Div(
			html.ID("question-area"),
			html.Class("question-area"),
			QuestionCard(sid, item, numResponses),
		),
		keyboardScript(),
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
			g.Attr("data-choice-num", fmt.Sprintf("%d", num)),
			hx.Post("/ui/respond"),
			hx.Target("#question-area"),
			hx.Swap("innerHTML"),
			hx.Vals(fmt.Sprintf(`{"sid":"%s","item_name":"%s","value":%d}`, sid, item.Name, c.Value.Value)),
			html.Span(html.Class("choice-number"), g.Textf("%d.", num)),
			html.Span(html.Class("choice-key-hint"), g.Textf("%d", num)),
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
		g.Attr("data-choice-num", "0"),
		hx.Post("/ui/respond"),
		hx.Target("#question-area"),
		hx.Swap("innerHTML"),
		hx.Vals(fmt.Sprintf(`{"sid":"%s","item_name":"%s","value":%d}`, sid, item.Name, skipValue)),
		g.Text("Skip"),
		html.Span(html.Class("choice-key-hint"), g.Text("S")),
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

// keyboardScript adds a keyboard listener for number keys → choice clicks.
func keyboardScript() g.Node {
	return html.Script(g.Raw(`
(function(){
  document.addEventListener("keydown", function(e) {
    if (e.target.tagName === "INPUT" || e.target.tagName === "TEXTAREA") return;
    var num = null;
    if (e.key >= "1" && e.key <= "9") num = e.key;
    else if (e.key === "0" || e.key === "s" || e.key === "S") num = "0";
    if (num === null) return;
    var btn = document.querySelector('[data-choice-num="' + num + '"]');
    if (btn) { btn.click(); e.preventDefault(); }
  });
})();
`))
}
