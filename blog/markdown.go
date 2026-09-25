package blog

import (
	"bytes"
	"html/template"
	"strconv"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
)

// Heading is one entry of a post's table of contents.
type Heading struct {
	ID    string
	Text  string
	Level int
}

type Rendered struct {
	HTML     template.HTML
	Headings []Heading
}

// Renderer turns a post's Markdown into HTML.
//
// Raw HTML in the Markdown is passed through: posts embed videos and live
// demos, and they come from this site's own CMS, which is trusted to the same
// degree as the templates.
type Renderer struct {
	md     goldmark.Markdown
	client *Client
}

func NewRenderer(client *Client) *Renderer {
	return &Renderer{
		md: goldmark.New(
			goldmark.WithExtensions(
				extension.GFM,
				highlighting.NewHighlighting(
					highlighting.WithStyle("dracula"),
					highlighting.WithFormatOptions(chromahtml.TabWidth(4)),
				),
			),
			goldmark.WithParserOptions(parser.WithAutoHeadingID()),
			goldmark.WithRendererOptions(html.WithUnsafe()),
		),
		client: client,
	}
}

// Render converts source, collecting its level 2 and 3 headings.
func (r *Renderer) Render(source string) (Rendered, error) {
	src := []byte(source)
	doc := r.md.Parser().Parse(text.NewReader(src))

	var headings []Heading
	err := ast.Walk(doc, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.Heading:
			if n.Level < 2 || n.Level > 3 {
				return ast.WalkContinue, nil
			}
			id, _ := n.AttributeString("id")
			if value, ok := id.([]byte); ok {
				headings = append(headings, Heading{ID: string(value), Text: plainText(n, src), Level: n.Level})
			}
		case *ast.Image:
			n.Destination = []byte(r.client.Asset(string(n.Destination)))
			// Declared, so opti-image serves it from this site.
			if r.client.IsAsset(string(n.Destination)) {
				n.SetAttributeString("width", []byte(strconv.Itoa(ImageWidth)))
				n.SetAttributeString("height", []byte(strconv.Itoa(ImageHeight)))
				n.SetAttributeString("loading", []byte("lazy"))
			}
		case *ast.Link:
			if strings.HasPrefix(string(n.Destination), "/uploads/") {
				n.Destination = []byte(r.client.Asset(string(n.Destination)))
			}
		}
		return ast.WalkContinue, nil
	})
	if err != nil {
		return Rendered{}, err
	}

	var out bytes.Buffer
	if err := r.md.Renderer().Render(&out, src, doc); err != nil {
		return Rendered{}, err
	}
	return Rendered{HTML: template.HTML(out.String()), Headings: headings}, nil
}

// plainText is a heading's text without its markup: "never trust `alg`" reads
// "never trust alg".
func plainText(node ast.Node, src []byte) string {
	var b strings.Builder
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := n.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(src))
		case *ast.String:
			b.Write(t.Value)
		}
		return ast.WalkContinue, nil
	})
	return b.String()
}
