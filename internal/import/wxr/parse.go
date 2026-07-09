// Package wxr parses WordPress WXR (eXtended RSS) export files.
package wxr

import (
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Export is the parsed contents of a WXR file.
type Export struct {
	BaseURL     string
	Attachments map[int]string // post_id -> attachment URL
	Posts       []Post
}

// Post is a WordPress post item from a WXR export.
type Post struct {
	ID           int
	Title        string
	Slug         string
	Content      string
	Excerpt      string
	Status       string
	PostType     string
	PublishedAt  time.Time
	Tags         []string
	ThumbnailID  int
}

type rss struct {
	Channel channel `xml:"channel"`
}

type channel struct {
	BaseURL string `xml:"http://wordpress.org/export/1.2/ base_blog_url"`
	Items   []item `xml:"item"`
}

type item struct {
	Title        string     `xml:"title"`
	Content      string     `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	Excerpt      string     `xml:"http://wordpress.org/export/1.2/excerpt/ encoded"`
	Categories   []category `xml:"category"`
	PostID       int        `xml:"http://wordpress.org/export/1.2/ post_id"`
	PostName     string     `xml:"http://wordpress.org/export/1.2/ post_name"`
	Status       string     `xml:"http://wordpress.org/export/1.2/ status"`
	PostType     string     `xml:"http://wordpress.org/export/1.2/ post_type"`
	PostDate     string     `xml:"http://wordpress.org/export/1.2/ post_date"`
	PostDateGMT  string     `xml:"http://wordpress.org/export/1.2/ post_date_gmt"`
	AttachmentURL string    `xml:"http://wordpress.org/export/1.2/ attachment_url"`
	PostMeta     []postMeta `xml:"http://wordpress.org/export/1.2/ postmeta"`
}

type category struct {
	Domain string `xml:"domain,attr"`
	Value  string `xml:",chardata"`
}

type postMeta struct {
	Key   string `xml:"http://wordpress.org/export/1.2/ meta_key"`
	Value string `xml:"http://wordpress.org/export/1.2/ meta_value"`
}

// ParseFile reads and parses a WXR export from path.
func ParseFile(path string) (*Export, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse reads and parses a WXR export from r.
func Parse(r io.Reader) (*Export, error) {
	var doc rss
	dec := xml.NewDecoder(r)
	// WXR files can be large; default token limit is fine.
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("decode WXR: %w", err)
	}

	out := &Export{
		BaseURL:     strings.TrimSpace(doc.Channel.BaseURL),
		Attachments: map[int]string{},
	}

	for _, it := range doc.Channel.Items {
		pt := strings.TrimSpace(it.PostType)
		if pt == "attachment" {
			if it.PostID > 0 && strings.TrimSpace(it.AttachmentURL) != "" {
				out.Attachments[it.PostID] = strings.TrimSpace(it.AttachmentURL)
			}
			continue
		}
		if pt != "post" {
			continue
		}

		dateStr := strings.TrimSpace(it.PostDateGMT)
		if dateStr == "" {
			dateStr = strings.TrimSpace(it.PostDate)
		}
		pub, err := parseWPDate(dateStr)
		if err != nil {
			pub = time.Time{}
		}

		p := Post{
			ID:          it.PostID,
			Title:       strings.TrimSpace(it.Title),
			Slug:        strings.TrimSpace(it.PostName),
			Content:     it.Content,
			Excerpt:     strings.TrimSpace(stripTags(it.Excerpt)),
			Status:      strings.TrimSpace(it.Status),
			PostType:    pt,
			PublishedAt: pub,
		}
		for _, c := range it.Categories {
			if c.Domain == "post_tag" {
				if v := strings.TrimSpace(c.Value); v != "" {
					p.Tags = append(p.Tags, v)
				}
			}
		}
		for _, m := range it.PostMeta {
			if strings.TrimSpace(m.Key) == "_thumbnail_id" {
				if id, err := strconv.Atoi(strings.TrimSpace(m.Value)); err == nil {
					p.ThumbnailID = id
				}
			}
		}
		out.Posts = append(out.Posts, p)
	}
	return out, nil
}

func parseWPDate(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty date")
	}
	return time.ParseInLocation("2006-01-02 15:04:05", s, time.UTC)
}

func stripTags(s string) string {
	// excerpt is usually plain text; quick strip for stray tags
	for strings.Contains(s, "<") {
		start := strings.Index(s, "<")
		end := strings.Index(s[start:], ">")
		if end < 0 {
			break
		}
		s = s[:start] + s[start+end+1:]
	}
	return s
}
