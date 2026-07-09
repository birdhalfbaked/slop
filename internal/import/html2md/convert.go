// Package html2md converts cleaned semantic HTML into Markdown with HTML figure blocks.
package html2md

import (
	"fmt"
	stdhtml "html"
	"strings"

	"golang.org/x/net/html"
)
// Convert turns cleaned HTML into mostly-Markdown with embedded figure blocks.
// rewriteURL maps remote image URLs to local paths when non-nil.
func Convert(cleaned string, rewriteURL func(string) string) (string, error) {
	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "", nil
	}

	doc, err := html.Parse(strings.NewReader("<body>" + cleaned + "</body>"))
	if err != nil {
		return "", fmt.Errorf("parse html: %w", err)
	}
	body := findChild(doc, "body")
	if body == nil {
		return "", nil
	}

	var blocks []string
	for c := body.FirstChild; c != nil; c = c.NextSibling {
		if s := renderBlock(c, rewriteURL); strings.TrimSpace(s) != "" {
			blocks = append(blocks, s)
		}
	}
	return strings.TrimSpace(strings.Join(blocks, "\n\n")), nil
}

func renderBlock(n *html.Node, rewriteURL func(string) string) string {
	if n.Type != html.ElementNode {
		return ""
	}
	tag := strings.ToLower(n.Data)
	switch {
	case tag == "figure":
		return renderFigure(n, rewriteURL)
	case strings.HasPrefix(tag, "h") && len(tag) == 2 && tag[1] >= '1' && tag[1] <= '6':
		level := int(tag[1] - '0')
		return strings.Repeat("#", level) + " " + strings.TrimSpace(renderInline(n))
	case tag == "p":
		return strings.TrimSpace(renderInline(n))
	case tag == "pre":
		code := textContent(n)
		code = strings.TrimRight(code, "\n")
		return "```\n" + code + "\n```"
	case tag == "ul":
		return renderList(n, false, 0)
	case tag == "ol":
		return renderList(n, true, 0)
	case tag == "blockquote":
		return renderBlockquote(n)
	case tag == "hr":
		return "---"
	default:
		return strings.TrimSpace(renderInline(n))
	}
}

func renderList(n *html.Node, ordered bool, depth int) string {
	var lines []string
	num := 1
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode || strings.ToLower(c.Data) != "li" {
			continue
		}
		main, nested := renderListItem(c, ordered, depth, num)
		if main != "" {
			lines = append(lines, main)
		}
		if nested != "" {
			lines = append(lines, nested)
		}
		if ordered {
			num++
		}
	}
	return strings.Join(lines, "\n")
}

func renderListItem(n *html.Node, ordered bool, depth int, num int) (string, string) {
	indent := strings.Repeat("  ", depth)
	marker := "-"
	if ordered {
		marker = fmt.Sprintf("%d.", num)
	}
	var inline strings.Builder
	var nested []string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			if c.Type == html.TextNode {
				inline.WriteString(escapeMarkdown(strings.TrimSpace(c.Data)))
			}
			continue
		}
		tag := strings.ToLower(c.Data)
		switch tag {
		case "ul":
			nested = append(nested, renderList(c, false, depth+1))
		case "ol":
			nested = append(nested, renderList(c, true, depth+1))
		default:
			inline.WriteString(renderInline(c))
		}
	}
	line := indent + marker + " " + strings.TrimSpace(inline.String())
	return line, strings.Join(nested, "\n")
}

func renderBlockquote(n *html.Node) string {
	var lines []string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		tag := strings.ToLower(c.Data)
		switch tag {
		case "p":
			text := strings.TrimSpace(renderInline(c))
			if text != "" {
				lines = append(lines, "> "+text)
			}
		case "cite":
			text := strings.TrimSpace(renderInline(c))
			if text != "" {
				lines = append(lines, ">")
				lines = append(lines, "> — *"+text+"*")
			}
		default:
			text := strings.TrimSpace(renderInline(c))
			if text != "" {
				lines = append(lines, "> "+text)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func renderFigure(n *html.Node, rewriteURL func(string) string) string {
	var src, alt, caption string
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type != html.ElementNode {
			continue
		}
		switch strings.ToLower(c.Data) {
		case "img":
			src = attrVal(c, "src")
			alt = attrVal(c, "alt")
		case "figcaption":
			caption = strings.TrimSpace(textContent(c))
		}
	}
	if rewriteURL != nil && src != "" {
		src = rewriteURL(src)
	}
	if src == "" {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<figure class="post-figure">` + "\n")
	b.WriteString(`  <img src="` + stdhtml.EscapeString(src) + `"`)
	if alt != "" {
		b.WriteString(` alt="` + stdhtml.EscapeString(alt) + `"`)
	}
	b.WriteString(">\n")
	if caption != "" {
		b.WriteString("  <figcaption>" + stdhtml.EscapeString(caption) + "</figcaption>\n")
	}
	b.WriteString("</figure>")
	return b.String()
}

func renderInline(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		switch node.Type {
		case html.TextNode:
			b.WriteString(escapeMarkdown(node.Data))
		case html.ElementNode:
			tag := strings.ToLower(node.Data)
			switch tag {
			case "strong", "b":
				b.WriteString("**")
				for c := node.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				b.WriteString("**")
			case "em", "i":
				b.WriteString("*")
				for c := node.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
				b.WriteString("*")
			case "code":
				b.WriteString("`")
				b.WriteString(strings.ReplaceAll(textContent(node), "`", "\\`"))
				b.WriteString("`")
			case "a":
				href := attrVal(node, "href")
				label := textContent(node)
				if href == "" {
					b.WriteString(escapeMarkdown(label))
				} else {
					b.WriteString("[" + strings.ReplaceAll(label, "]", "\\]") + "](" + href + ")")
				}
			case "br":
				b.WriteString("\n")
			case "sub", "sup", "u", "mark":
				b.WriteString("<" + tag + ">")
				for ch := node.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				b.WriteString("</" + tag + ">")
			case "del", "s", "strike":
				b.WriteString("~~")
				for ch := node.FirstChild; ch != nil; ch = ch.NextSibling {
					walk(ch)
				}
				b.WriteString("~~")
			default:
				for c := node.FirstChild; c != nil; c = c.NextSibling {
					walk(c)
				}
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walk(c)
	}
	return b.String()
}

func escapeMarkdown(s string) string {
	replacer := strings.NewReplacer(
		"\\", "\\\\",
		"*", "\\*",
		"_", "\\_",
		"`", "\\`",
		"[", "\\[",
	)
	return replacer.Replace(s)
}

func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}
	return ""
}

func textContent(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.TextNode {
			b.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

func findChild(n *html.Node, tag string) *html.Node {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.ElementNode && strings.ToLower(c.Data) == tag {
			return c
		}
		if sub := findChild(c, tag); sub != nil {
			return sub
		}
	}
	return nil
}
