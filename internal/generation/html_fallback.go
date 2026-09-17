package generation

import (
	"html"
	"strings"
)

func fallbackHTML(draft string) string {
	var body strings.Builder
	inList := false
	closeList := func() {
		if inList {
			body.WriteString("</ul>")
			inList = false
		}
	}
	for _, raw := range strings.Split(strings.ReplaceAll(draft, "\r\n", "\n"), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			closeList()
			continue
		}
		escaped := html.EscapeString(line)
		switch {
		case strings.HasPrefix(line, "### "):
			closeList()
			body.WriteString("<h3>" + html.EscapeString(strings.TrimSpace(line[4:])) + "</h3>")
		case strings.HasPrefix(line, "## "):
			closeList()
			body.WriteString("<h2>" + html.EscapeString(strings.TrimSpace(line[3:])) + "</h2>")
		case strings.HasPrefix(line, "# "):
			closeList()
			body.WriteString("<h1>" + html.EscapeString(strings.TrimSpace(line[2:])) + "</h1>")
		case strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* "):
			if !inList {
				body.WriteString("<ul>")
				inList = true
			}
			body.WriteString("<li>" + html.EscapeString(strings.TrimSpace(line[2:])) + "</li>")
		default:
			closeList()
			body.WriteString("<p>" + escaped + "</p>")
		}
	}
	closeList()
	return `<!doctype html><html lang="en"><head><meta charset="utf-8"><style>
@page { size: A4; margin: 16mm 18mm; }
* { box-sizing: border-box; }
body { margin: 0; color: #172033; font-family: Arial, Helvetica, sans-serif; font-size: 10pt; line-height: 1.42; }
main { max-width: 100%; }
h1 { margin: 0 0 5mm; color: #172033; font-size: 25pt; line-height: 1.05; letter-spacing: -.4pt; }
h2 { margin: 5mm 0 2mm; padding-bottom: 1.2mm; border-bottom: 1.2pt solid #4f6f8f; color: #234767; font-size: 12pt; text-transform: uppercase; letter-spacing: .7pt; break-after: avoid; }
h3 { margin: 3mm 0 1mm; color: #172033; font-size: 10.5pt; break-after: avoid; }
p { margin: 0 0 2.2mm; }
ul { margin: 1mm 0 2.5mm; padding-left: 5mm; }
li { margin-bottom: 1mm; }
h1, h2, h3, p, li { overflow-wrap: anywhere; }
li, h3 { break-inside: avoid; }
</style></head><body><main>` + body.String() + `</main></body></html>`
}
