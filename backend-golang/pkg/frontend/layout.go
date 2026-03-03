package frontend

import (
	g "maragu.dev/gomponents"
	c "maragu.dev/gomponents/components"
	"maragu.dev/gomponents/html"
)

func Page(title string, body ...g.Node) g.Node {
	return c.HTML5(c.HTML5Props{
		Title:    title + " — GoFlutterCat",
		Language: "en",
		Head: []g.Node{
			html.Meta(html.Name("viewport"), html.Content("width=device-width, initial-scale=1")),
			html.Meta(html.Name("color-scheme"), html.Content("light dark")),
			html.Link(html.Rel("stylesheet"), html.Href("https://cdn.jsdelivr.net/npm/@picocss/pico@2/css/pico.min.css")),
			html.Script(html.Src("https://unpkg.com/htmx.org@2.0.4")),
			html.StyleEl(g.Raw(customCSS)),
		},
		Body: []g.Node{
			html.Main(
				html.Class("container"),
				g.Group(body),
			),
		},
	})
}
