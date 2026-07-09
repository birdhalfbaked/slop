// Package htmlclean strips WordPress/Gutenberg HTML down to a minimal semantic subset.
package htmlclean

import (
	"bytes"
	"fmt"
	"html"
	"regexp"
	"strings"

	xhtml "golang.org/x/net/html"
)

var (
	gutenbergComment = regexp.MustCompile(`<!--\s*/?wp:[^>]*-->`)
	captionShortcode = regexp.MustCompile(`(?is)\[caption[^\]]*\](.*?)\[/caption\]`)
	shortcodeBlock   = regexp.MustCompile(`(?is)\[[a-z0-9_-]+[^\]]*\].*?\[/[a-z0-9_-]+\]`)
	shortcodeSelf    = regexp.MustCompile(`(?is)\[[a-z0-9_-]+[^\]]*\]`)
)

var blockTags = map[string]bool{
	"p": true, "h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
	"pre": true, "figure": true, "ul": true, "ol": true, "blockquote": true,
}

var inlineTags = map[string]bool{
	"a": true, "strong": true, "b": true, "em": true, "i": true, "code": true, "br": true,
	"sub": true, "sup": true, "del": true, "s": true, "strike": true, "u": true, "mark": true,
}

// Clean preprocesses WordPress HTML and rebuilds a minimal semantic fragment.
func Clean(src string) (string, error) {
	src = preprocess(src)
	if strings.TrimSpace(src) == "" {
		return "", nil
	}

	doc, err := xhtml.Parse(strings.NewReader("<body>" + src + "</body>"))
	if err != nil {
		return "", fmt.Errorf("parse html: %w", err)
	}
	body := findChild(doc, "body")
	if body == nil {
		return "", nil
	}

	var blocks []*xhtml.Node
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		blocks = append(blocks, collectBlocks(c)...)
	}
	blocks = wrapBareImages(blocks)

	var buf bytes.Buffer
	for i, n := range blocks {
		if i > 0 {
			buf.WriteString("\n")
		}
		if err := xhtml.Render(&buf, n); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func preprocess(s string) string {
	s = gutenbergComment.ReplaceAllString(s, "")
	s = captionShortcode.ReplaceAllStringFunc(s, convertCaption)
	s = shortcodeBlock.ReplaceAllString(s, "")
	s = shortcodeSelf.ReplaceAllString(s, "")
	return s
}

func convertCaption(match string) string {
	sub := captionShortcode.FindStringSubmatch(match)
	if len(sub) < 2 {
		return ""
	}
	inner := strings.TrimSpace(sub[1])
	imgRe := regexp.MustCompile(`(?is)<img[^>]*>`)
	img := imgRe.FindString(inner)
	if img == "" {
		return inner
	}
	caption := strings.TrimSpace(imgRe.ReplaceAllString(inner, ""))
	if caption == "" {
		return "<figure>" + img + "</figure>"
	}
	return "<figure>" + img + "<figcaption>" + html.EscapeString(caption) + "</figcaption></figure>"
}

func collectBlocks(n *xhtml.Node) []*xhtml.Node {
	if n.Type == xhtml.TextNode {
		text := strings.TrimSpace(n.Data)
		if text == "" {
			return nil
		}
		return []*xhtml.Node{newElement("p", nil, textNode(text))}
	}
	if n.Type != xhtml.ElementNode {
		return nil
	}

	tag := strings.ToLower(n.Data)
	if blockTags[tag] {
		return []*xhtml.Node{cleanBlock(n)}
	}
	if tag == "img" {
		return []*xhtml.Node{figureFromImg(cleanImg(n))}
	}
	if tag == "hr" {
		return []*xhtml.Node{newElement("hr", nil)}
	}

	// Unwrap divs, groups, etc. — keep child content as blocks.
	var out []*xhtml.Node
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		out = append(out, collectBlocks(c)...)
	}
	return out
}

func cleanBlock(n *xhtml.Node) *xhtml.Node {
	tag := strings.ToLower(n.Data)
	switch tag {
	case "figure":
		return cleanFigure(n)
	case "pre":
		return cleanPre(n)
	case "ul", "ol":
		return cleanList(n)
	case "blockquote":
		return cleanBlockquote(n)
	default:
		el := newElement(tag, allowedBlockAttrs(n))
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			appendInlineChildren(el, c)
		}
		return el
	}
}

func cleanList(n *xhtml.Node) *xhtml.Node {
	list := newElement(strings.ToLower(n.Data), nil)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == xhtml.ElementNode && strings.ToLower(c.Data) == "li" {
			list.AppendChild(cleanListItem(c))
		}
	}
	return list
}

func cleanListItem(n *xhtml.Node) *xhtml.Node {
	li := newElement("li", nil)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == xhtml.ElementNode {
			tag := strings.ToLower(c.Data)
			switch tag {
			case "ul", "ol":
				li.AppendChild(cleanList(c))
			case "p":
				for pc := c.FirstChild; pc != nil; pc = pc.NextSibling {
					appendInlineChildren(li, pc)
				}
			default:
				appendInlineChildren(li, c)
			}
		} else if c.Type == xhtml.TextNode {
			appendInlineChildren(li, c)
		}
	}
	return li
}

func cleanBlockquote(n *xhtml.Node) *xhtml.Node {
	bq := newElement("blockquote", nil)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != xhtml.ElementNode {
			continue
		}
		tag := strings.ToLower(c.Data)
		switch tag {
		case "p":
			p := newElement("p", nil)
			for pc := c.FirstChild; pc != nil; pc = pc.NextSibling {
				appendInlineChildren(p, pc)
			}
			bq.AppendChild(p)
		case "cite":
			bq.AppendChild(newElement("cite", nil, textNode(textContent(c))))
		default:
			for child := c.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == xhtml.ElementNode && strings.ToLower(child.Data) == "p" {
					p := newElement("p", nil)
					for pc := child.FirstChild; pc != nil; pc = pc.NextSibling {
						appendInlineChildren(p, pc)
					}
					bq.AppendChild(p)
				} else {
					appendInlineChildren(bq, child)
				}
			}
		}
	}
	return bq
}

func cleanFigure(n *xhtml.Node) *xhtml.Node {
	fig := newElement("figure", nil)
	var img *xhtml.Node
	var caption string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == xhtml.ElementNode && strings.ToLower(c.Data) == "img" {
			img = cleanImg(c)
		}
		if c.Type == xhtml.ElementNode && strings.ToLower(c.Data) == "figcaption" {
			caption = textContent(c)
		}
	}
	if img == nil {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == xhtml.ElementNode && strings.ToLower(c.Data) == "img" {
				img = cleanImg(c)
			}
		}
	}
	if img != nil {
		fig.AppendChild(img)
	}
	if strings.TrimSpace(caption) != "" {
		fig.AppendChild(newElement("figcaption", nil, textNode(caption)))
	}
	return fig
}

func cleanPre(n *xhtml.Node) *xhtml.Node {
	pre := newElement("pre", nil)
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == xhtml.ElementNode && strings.ToLower(c.Data) == "code" {
			pre.AppendChild(newElement("code", nil, textNode(textContent(c))))
			return pre
		}
	}
	pre.AppendChild(textNode(textContent(n)))
	return pre
}

func cleanImg(n *xhtml.Node) *xhtml.Node {
	el := newElement("img", nil)
	for _, a := range n.Attr {
		k := strings.ToLower(a.Key)
		if k == "src" || k == "alt" {
			el.Attr = append(el.Attr, xhtml.Attribute{Key: k, Val: a.Val})
		}
	}
	return el
}

func figureFromImg(img *xhtml.Node) *xhtml.Node {
	fig := newElement("figure", nil)
	fig.AppendChild(img)
	return fig
}

func wrapBareImages(blocks []*xhtml.Node) []*xhtml.Node {
	out := make([]*xhtml.Node, 0, len(blocks))
	for _, b := range blocks {
		if b.Type == xhtml.ElementNode && strings.ToLower(b.Data) == "img" {
			out = append(out, figureFromImg(b))
			continue
		}
		out = append(out, b)
	}
	return out
}

func appendInlineChildren(parent *xhtml.Node, n *xhtml.Node) {
	if n.Type == xhtml.TextNode {
		if strings.TrimSpace(n.Data) != "" {
			parent.AppendChild(textNode(n.Data))
		}
		return
	}
	if n.Type != xhtml.ElementNode {
		return
	}
	tag := strings.ToLower(n.Data)
	if inlineTags[tag] {
		el := newElement(tag, allowedInlineAttrs(n))
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type == xhtml.TextNode {
				el.AppendChild(textNode(c.Data))
			} else if c.Type == xhtml.ElementNode {
				appendInlineChildren(el, c)
			}
		}
		parent.AppendChild(el)
		return
	}
	// Unwrap unknown inline/block inside paragraph.
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		appendInlineChildren(parent, c)
	}
}

func allowedBlockAttrs(n *xhtml.Node) []xhtml.Attribute { return nil }

func allowedInlineAttrs(n *xhtml.Node) []xhtml.Attribute {
	tag := strings.ToLower(n.Data)
	if tag != "a" {
		return nil
	}
	for _, a := range n.Attr {
		if strings.ToLower(a.Key) == "href" {
			return []xhtml.Attribute{{Key: "href", Val: a.Val}}
		}
	}
	return nil
}

func newElement(tag string, attrs []xhtml.Attribute, children ...*xhtml.Node) *xhtml.Node {
	n := &xhtml.Node{Type: xhtml.ElementNode, Data: tag, Attr: attrs}
	for _, c := range children {
		if c != nil {
			n.AppendChild(c)
		}
	}
	return n
}

func textNode(s string) *xhtml.Node {
	return &xhtml.Node{Type: xhtml.TextNode, Data: s}
}

func textContent(n *xhtml.Node) string {
	var b strings.Builder
	var walk func(*xhtml.Node)
	walk = func(node *xhtml.Node) {
		if node.Type == xhtml.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func findChild(n *xhtml.Node, tag string) *xhtml.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == xhtml.ElementNode && strings.ToLower(c.Data) == tag {
			return c
		}
		if sub := findChild(c, tag); sub != nil {
			return sub
		}
	}
	return nil
}
