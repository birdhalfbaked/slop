package render

import (
	"bytes"
	"database/sql"
	"fmt"
	"html"
	"html/template"
	"io"
	"regexp"
	"strings"
	"time"

	"slop/internal/models"
	"slop/internal/theme"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	ghtml "github.com/yuin/goldmark/renderer/html"
)

var md = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(ghtml.WithUnsafe()),
)

var htmlTagRe = regexp.MustCompile(`(?s)<[^>]*>`)

// RenderMarkdown converts markdown source to HTML.
func RenderMarkdown(src string) (string, error) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// RenderBody renders a post's source into HTML based on its format.
func RenderBody(format, src string) (string, error) {
	if format == "html" {
		return src, nil
	}
	return RenderMarkdown(src)
}

// StripHTML removes HTML tags and unescapes entities for plain-text contexts (meta, alt).
func StripHTML(s string) string {
	s = htmlTagRe.ReplaceAllString(s, "")
	return strings.TrimSpace(html.UnescapeString(s))
}

// SiteData holds site-wide values exposed to theme templates.
type SiteData struct {
	Title         string
	Description   string
	BaseURL       string
	OGSiteName    string
	TwitterHandle string
}

// PostView is the view model exposed to theme templates for a post.
type PostView struct {
	ID          int64
	Date        time.Time
	Title       string
	TitleHTML   template.HTML
	TitlePlain  string
	Slug        string
	Image       string
	Tags        []string
	Content     template.HTML
	Description string
	Status      string
}

// PageOpts carries optional pagination and tag-archive context for public pages.
type PageOpts struct {
	Tag        string
	TagSlug    string
	Page       int
	TotalPages int
	HasPrev    bool
	HasNext    bool
	PrevPage   int
	NextPage   int
}

// StaticPageView is the view model exposed to theme templates for a static page.
type StaticPageView struct {
	ID          int64
	Title       string
	TitleHTML   template.HTML
	TitlePlain  string
	Slug        string
	Content     template.HTML
	Status      string
}

// PageData is the root data passed to theme templates.
type PageData struct {
	Site        SiteData
	Post        *PostView
	StaticPage  *StaticPageView
	Posts       []*PostView
	Tag        string
	TagSlug    string
	Page       int
	TotalPages int
	HasPrev    bool
	HasNext    bool
	PrevPage   int
	NextPage   int
	Content    template.HTML
	Meta       template.HTML
	Fonts      template.HTML
	Styles     template.HTML
	Scripts    template.HTML
}

// Renderer renders public pages from DB-stored theme fragments.
type Renderer struct {
	db *sql.DB
}

func New(db *sql.DB) *Renderer { return &Renderer{db: db} }

// SetDB updates the database used for rendering after a restore.
func (r *Renderer) SetDB(db *sql.DB) { r.db = db }

var funcs = template.FuncMap{
	"formatDate": func(t time.Time, layout string) string { return t.Format(layout) },
	"safeHTML":   func(s string) template.HTML { return template.HTML(s) },
	"join":       strings.Join,
	"tagSlug":    models.TagSlug,
}

func (r *Renderer) buildTemplateSet(settings map[string]string) (*template.Template, error) {
	root := template.New("root").Funcs(funcs)
	for _, f := range theme.Fragments {
		if f.Language != "html" {
			continue
		}
		if _, err := root.New(f.Key).Parse(settings[f.Key]); err != nil {
			return nil, fmt.Errorf("parsing theme fragment %q: %w", f.Key, err)
		}
	}
	return root, nil
}

// ToPostView builds a view model from a stored post.
func ToPostView(p *models.Post) *PostView {
	date := p.CreatedAt
	if p.PublishedAt != nil {
		date = *p.PublishedAt
	}
	return &PostView{
		ID:          p.ID,
		Date:        date,
		Title:       p.Title,
		TitleHTML:   template.HTML(p.Title),
		TitlePlain:  StripHTML(p.Title),
		Slug:        p.Slug,
		Image:       p.Image,
		Tags:        p.Tags,
		Content:     template.HTML(p.RenderedHTML),
		Description: p.Description,
		Status:      p.Status,
	}
}

// ToStaticPageView builds a view model from a stored static page.
func ToStaticPageView(p *models.StaticPage) *StaticPageView {
	return &StaticPageView{
		ID:         p.ID,
		Title:      p.Title,
		TitleHTML:  template.HTML(p.Title),
		TitlePlain: StripHTML(p.Title),
		Slug:       p.Slug,
		Content:    template.HTML(p.RenderedHTML),
		Status:     p.Status,
	}
}

// RenderPage renders the named page template wrapped in the base layout.
func (r *Renderer) RenderPage(w io.Writer, pageTemplate string, post *models.Post, posts []*models.Post, opts *PageOpts, staticPage *models.StaticPage) error {
	settings, err := models.GetAllSettings(r.db)
	if err != nil {
		return err
	}
	fonts, err := models.ListFonts(r.db)
	if err != nil {
		return err
	}
	set, err := r.buildTemplateSet(settings)
	if err != nil {
		return err
	}

	data := &PageData{
		Site: SiteData{
			Title:         settings["site_title"],
			Description:   settings["site_description"],
			BaseURL:       strings.TrimRight(settings["base_url"], "/"),
			OGSiteName:    settings["og_site_name"],
			TwitterHandle: settings["twitter_handle"],
		},
		Meta:    buildMeta(settings, post, staticPage),
		Fonts:   buildFonts(fonts),
		Styles:  template.HTML("<style>\n" + settings["theme_css"] + "\n</style>"),
		Scripts: template.HTML("<script>\n" + settings["theme_js"] + "\n</script>"),
	}
	if opts != nil {
		data.Tag = opts.Tag
		data.TagSlug = opts.TagSlug
		data.Page = opts.Page
		data.TotalPages = opts.TotalPages
		data.HasPrev = opts.HasPrev
		data.HasNext = opts.HasNext
		data.PrevPage = opts.PrevPage
		data.NextPage = opts.NextPage
	}
	if post != nil {
		data.Post = ToPostView(post)
	}
	if staticPage != nil {
		data.StaticPage = ToStaticPageView(staticPage)
	}
	for _, p := range posts {
		data.Posts = append(data.Posts, ToPostView(p))
	}

	var inner bytes.Buffer
	if err := set.ExecuteTemplate(&inner, pageTemplate, data); err != nil {
		return err
	}
	data.Content = template.HTML(inner.String())
	return set.ExecuteTemplate(w, "base_layout", data)
}

func resolve(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func metaTag(name, content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	return fmt.Sprintf("<meta name=%q content=%q>\n", name, html.EscapeString(content))
}

func ogTag(property, content string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	return fmt.Sprintf("<meta property=%q content=%q>\n", property, html.EscapeString(content))
}

func buildMeta(settings map[string]string, post *models.Post, staticPage *models.StaticPage) template.HTML {
	siteTitle := settings["site_title"]
	var b strings.Builder

	var title, description, image, url, ogType string
	if post != nil {
		title = StripHTML(resolve(post.OGTitle, post.Title, siteTitle))
		description = resolve(post.MetaDescription, post.OGDescription, post.Description, settings["site_description"])
		image = resolve(post.OGImage, post.Image, settings["og_default_image"])
		if base := strings.TrimRight(settings["base_url"], "/"); base != "" {
			url = base + "/" + post.Slug
		}
		ogType = "article"
	} else if staticPage != nil {
		title = StripHTML(resolve(staticPage.Title, siteTitle))
		description = settings["site_description"]
		image = settings["og_default_image"]
		if base := strings.TrimRight(settings["base_url"], "/"); base != "" {
			url = base + "/" + staticPage.Slug
		}
		ogType = "website"
	} else {
		title = StripHTML(siteTitle)
		description = settings["site_description"]
		image = settings["og_default_image"]
		url = strings.TrimRight(settings["base_url"], "/")
		ogType = "website"
	}

	b.WriteString(fmt.Sprintf("<title>%s</title>\n", html.EscapeString(title)))
	b.WriteString(metaTag("description", description))

	b.WriteString(ogTag("og:title", title))
	b.WriteString(ogTag("og:description", description))
	b.WriteString(ogTag("og:type", ogType))
	b.WriteString(ogTag("og:image", image))
	b.WriteString(ogTag("og:url", url))
	b.WriteString(ogTag("og:site_name", settings["og_site_name"]))

	b.WriteString(metaTag("twitter:card", "summary_large_image"))
	b.WriteString(metaTag("twitter:title", title))
	b.WriteString(metaTag("twitter:description", description))
	b.WriteString(metaTag("twitter:image", image))
	b.WriteString(metaTag("twitter:site", settings["twitter_handle"]))

	return template.HTML(b.String())
}

func buildFonts(fonts []*models.Font) template.HTML {
	var links strings.Builder
	var faces strings.Builder
	for _, f := range fonts {
		if f.Source == "cdn" {
			if strings.TrimSpace(f.URL) == "" {
				continue
			}
			links.WriteString(fmt.Sprintf("<link rel=\"stylesheet\" href=%q>\n", html.EscapeString(f.URL)))
			continue
		}
		if strings.TrimSpace(f.Filename) == "" {
			continue
		}
		family := f.Family
		if family == "" {
			family = f.Name
		}
		faces.WriteString("@font-face {\n")
		faces.WriteString(fmt.Sprintf("  font-family: %q;\n", family))
		faces.WriteString(fmt.Sprintf("  src: url(\"/static/fonts/%s\");\n", f.Filename))
		if f.Weight != "" {
			faces.WriteString(fmt.Sprintf("  font-weight: %s;\n", f.Weight))
		}
		if f.Style != "" {
			faces.WriteString(fmt.Sprintf("  font-style: %s;\n", f.Style))
		}
		faces.WriteString("  font-display: swap;\n}\n")
	}
	out := links.String()
	if faces.Len() > 0 {
		out += "<style>\n" + faces.String() + "</style>\n"
	}
	return template.HTML(out)
}
