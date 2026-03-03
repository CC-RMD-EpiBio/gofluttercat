package frontend

import (
	g "maragu.dev/gomponents"
	"maragu.dev/gomponents/html"
)

func Navbar() g.Node {
	return html.Nav(
		html.Ul(
			html.Li(html.A(html.Href("/"), html.Strong(g.Text("GoFlutterCat")))),
		),
		html.Ul(
			html.Li(html.A(html.Href("/docs"), g.Text("API Docs"))),
		),
	)
}

func ErrorAlert(msg string) g.Node {
	return html.Div(
		html.Role("alert"),
		html.Class("pico-background-red-500"),
		html.Style("padding:1rem;border-radius:var(--pico-border-radius);margin-bottom:1rem;color:white"),
		g.Text(msg),
	)
}

func LoadingIndicator() g.Node {
	return html.Span(
		html.Class("htmx-indicator"),
		g.Attr("aria-busy", "true"),
	)
}
