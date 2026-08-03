package transport

import (
	"context"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"

	"github.com/eefenaxce/axce_blog/internal/models"
	"github.com/eefenaxce/axce_blog/internal/service"
	thymeleaf "github.com/eefenaxce/axce_blog/pkg/thymeleaf-go"
)

// ThemeRenderer handles server-side theme template rendering.
type ThemeRenderer struct {
	mu              sync.RWMutex
	engine          *thymeleaf.Engine
	loader          *thymeleaf.FileSystemLoader
	activeThemeID   string
	messageResolver *thymeleaf.PropertiesMessageResolver
	themeService    *service.ThemeService
	settingService  *service.SettingService
	articleService  *service.ArticleService
	categoryService *service.CategoryService
	tagService      *service.TagService
	userService     *service.UserService
	menuService     *service.MenuService
	commentService  *service.CommentService
}

func NewThemeRenderer(
	themeService *service.ThemeService,
	settingService *service.SettingService,
	articleService *service.ArticleService,
	categoryService *service.CategoryService,
	tagService *service.TagService,
	userService *service.UserService,
	menuService *service.MenuService,
	commentService *service.CommentService,
) *ThemeRenderer {
	return &ThemeRenderer{
		themeService:    themeService,
		settingService:  settingService,
		articleService:  articleService,
		categoryService: categoryService,
		tagService:      tagService,
		userService:     userService,
		menuService:     menuService,
		commentService:  commentService,
	}
}

// ensureEngine lazily creates or updates the thymeleaf engine for the active theme.
func (r *ThemeRenderer) ensureEngine(ctx context.Context) error {
	theme, err := r.themeService.GetActiveTheme(ctx)
	if err != nil {
		return err
	}

	r.mu.RLock()
	currentID := r.activeThemeID
	r.mu.RUnlock()

	if r.engine != nil && currentID == theme.ID {
		return nil
	}

	// Theme changed (or first load) — create new engine
	themeDir, err := r.themeService.GetThemeDir(theme.ID)
	if err != nil {
		return err
	}

	templatesDir := filepath.Join(themeDir, "templates")
	loader := thymeleaf.NewFileSystemLoader(templatesDir)
	engine := thymeleaf.NewEngine(loader)

	// Load i18n message bundles from theme's i18n/ directory
	resolver := thymeleaf.NewPropertiesMessageResolver()
	i18nDir := filepath.Join(themeDir, "i18n")
	r.loadI18nFiles(resolver, i18nDir)

	r.mu.Lock()
	r.engine = engine
	r.loader = loader
	r.activeThemeID = theme.ID
	r.messageResolver = resolver
	r.mu.Unlock()

	return nil
}

// loadI18nFiles loads all .properties files from the i18n directory.
// "default.properties" is loaded first, then locale-specific files override.
func (r *ThemeRenderer) loadI18nFiles(resolver *thymeleaf.PropertiesMessageResolver, i18nDir string) {
	// Load default first
	defaultPath := filepath.Join(i18nDir, "default.properties")
	if _, err := os.Stat(defaultPath); err == nil {
		resolver.LoadFile(defaultPath)
	}

	// Try to load zh_CN (could be made locale-aware later)
	localePath := filepath.Join(i18nDir, "zh_CN.properties")
	if _, err := os.Stat(localePath); err == nil {
		resolver.LoadFile(localePath)
	}
}

// ServeHomepage handles GET / — renders the active theme's index.html template.
func (r *ThemeRenderer) ServeHomepage(c fiber.Ctx) error {
	page := parsePageQuery(c.Query("page"))
	return r.renderPage(c, "index", "index", page, nil)
}

// ServeHomepagePaged handles GET /page/:n — paginated homepage.
func (r *ThemeRenderer) ServeHomepagePaged(c fiber.Ctx) error {
	page := parsePageParam(c.Params("n"))
	return r.renderPage(c, "index", "index", page, nil)
}

// ServePost handles GET /archives/:slug — renders a single post.
func (r *ThemeRenderer) ServePost(c fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusNotFound).SendString("Post not found")
	}

	ctx := c.Context()
	article, err := r.articleService.GetBySlug(ctx, slug)
	if err != nil || article == nil {
		return c.Status(fiber.StatusNotFound).SendString("Post not found")
	}

	r.articleService.IncrementViewCount(ctx, article.ID)

	tags, _ := r.articleService.GetTags(ctx, article.ID)
	var author *models.User
	if u, err := r.userService.GetByID(ctx, article.UserID); err == nil {
		author = u
	}
	categories := r.getPostCategories(article.Categories)
	commentCounts, _ := r.commentService.CountByArticles(ctx, []int{article.ID})
	post := buildHaloPost(article, tags, author, categories, commentCounts[article.ID])

	postFinder := NewPostFinder(r.articleService, r.userService)
	cursor := postFinder.Cursor(article.Slug)

	comments, _, _ := r.commentService.ListByArticle(ctx, article.ID, 0, 1000)
	var commentList []map[string]any
	for _, cm := range comments {
		commentList = append(commentList, map[string]any{
			"metadata": map[string]any{
				"name": cm.AuthorName,
			},
			"spec": map[string]any{
				"displayName": cm.AuthorName,
				"email":       cm.AuthorEmail,
				"website":     cm.AuthorURL,
				"content":     cm.Content,
				"createTime":  cm.CreatedAt.Format("2006-01-02T15:04:05Z"),
				"approved":    cm.Status == "approved",
			},
			"status": map[string]any{
				"replyCount": 0,
			},
		})
	}
	if commentList == nil {
		commentList = []map[string]any{}
	}

	extra := map[string]any{
		"post":         post,
		"postCursor":   cursor,
		"comments":     commentList,
		"commentCount": len(commentList),
	}
	return r.renderPage(c, "post", "post", 1, extra)
}

// ServeArchives handles GET /archives — renders the archives page.
func (r *ThemeRenderer) ServeArchives(c fiber.Ctx) error {
	page := parsePageQuery(c.Query("page"))
	return r.renderArchives(c, page)
}

// ServeArchivesPaged handles GET /archives/page/:n.
func (r *ThemeRenderer) ServeArchivesPaged(c fiber.Ctx) error {
	page := parsePageParam(c.Params("n"))
	return r.renderArchives(c, page)
}

func (r *ThemeRenderer) renderArchives(c fiber.Ctx, page int) error {
	ctx := c.Context()
	if page < 1 {
		page = 1
	}
	pageSize := 10
	offset := (page - 1) * pageSize

	articles, total, err := r.articleService.PublicList(ctx, offset, pageSize, "", "")
	if err != nil {
		articles = nil
	}

	archives := r.buildArchivesObject(articles, total, page, pageSize, "/archives/")
	extra := map[string]any{"archives": archives}
	return r.renderPage(c, "archives", "archives", page, extra)
}

// ServeCategories handles GET /categories — renders the categories list page.
func (r *ThemeRenderer) ServeCategories(c fiber.Ctx) error {
	return r.renderPage(c, "categories", "categories", 1, nil)
}

// ServeCategory handles GET /categories/:slug — renders posts in a category.
func (r *ThemeRenderer) ServeCategory(c fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusNotFound).SendString("Category not found")
	}
	return r.renderCategory(c, slug, parsePageQuery(c.Query("page")))
}

// ServeCategoryPaged handles GET /categories/:slug/page/:n.
func (r *ThemeRenderer) ServeCategoryPaged(c fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusNotFound).SendString("Category not found")
	}
	return r.renderCategory(c, slug, parsePageParam(c.Params("n")))
}

func (r *ThemeRenderer) renderCategory(c fiber.Ctx, slug string, page int) error {
	ctx := c.Context()
	cat, err := r.categoryService.GetBySlug(ctx, slug)
	if err != nil || cat == nil {
		return c.Status(fiber.StatusNotFound).SendString("Category not found")
	}
	category := map[string]any{
		"spec": map[string]any{
			"displayName": cat.Name,
			"slug":        cat.Slug,
		},
		"metadata": map[string]any{
			"name":        cat.Slug,
			"displayName": cat.Name,
		},
		"status": map[string]any{
			"permalink":   "/categories/" + cat.Slug,
			"displayName": cat.Name,
		},
		"icon": cat.Icon,
	}

	if page < 1 {
		page = 1
	}
	pageSize := 10
	offset := (page - 1) * pageSize
	articles, total, err := r.articleService.PublicList(ctx, offset, pageSize, cat.Slug, "")

	commentCounts, _ := r.commentService.CountByArticles(ctx, articleIDs(articles))
	var items []any
	for _, a := range articles {
		tags, _ := r.articleService.GetTags(ctx, a.ID)
		var author *models.User
		if u, err := r.userService.GetByID(ctx, a.UserID); err == nil {
			author = u
		}
		categories := r.getPostCategories(a.Categories)
		items = append(items, buildHaloPost(a, tags, author, categories, commentCounts[a.ID]))
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	posts := &PostsResult{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		NextUrl:    nextPageURL(page, totalPages),
		PrevUrl:    prevPageURL(page),
	}

	extra := map[string]any{
		"category": category,
		"posts":    posts,
	}
	return r.renderPage(c, "category", "category", page, extra)
}

// ServeTags handles GET /tags — renders the tags list page.
func (r *ThemeRenderer) ServeTags(c fiber.Ctx) error {
	ctx := c.Context()
	tags, err := r.tagService.List(ctx)
	if err != nil {
		tags = nil
	}
	counts, _ := r.tagService.GetTagPostCounts(ctx)
	return r.renderPage(c, "tags", "tags", 1, map[string]any{
		"tags": buildHaloTagList(tags, counts),
	})
}

// ServeTag handles GET /tags/:slug — renders posts with a tag.
func (r *ThemeRenderer) ServeTag(c fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusNotFound).SendString("Tag not found")
	}
	return r.renderTag(c, slug, parsePageQuery(c.Query("page")))
}

// ServeTagPaged handles GET /tags/:slug/page/:n.
func (r *ThemeRenderer) ServeTagPaged(c fiber.Ctx) error {
	slug := c.Params("slug")
	if slug == "" {
		return c.Status(fiber.StatusNotFound).SendString("Tag not found")
	}
	return r.renderTag(c, slug, parsePageParam(c.Params("n")))
}

// ServeSearch handles GET /search — renders the index template with search results.
func (r *ThemeRenderer) ServeSearch(c fiber.Ctx) error {
	ctx := c.Context()
	keyword := strings.TrimSpace(c.Query("keyword"))

	page := 1
	pageSize := 10
	offset := 0

	var items []any
	var total int

	if keyword != "" {
		articles, count, err := r.articleService.Search(ctx, keyword, offset, pageSize)
		if err == nil {
			total = count
			commentCounts, _ := r.commentService.CountByArticles(ctx, articleIDs(articles))
			for _, a := range articles {
				tags, _ := r.articleService.GetTags(ctx, a.ID)
				var author *models.User
				if u, err := r.userService.GetByID(ctx, a.UserID); err == nil {
					author = u
				}
				categories := r.getPostCategories(a.Categories)
				items = append(items, buildHaloPost(a, tags, author, categories, commentCounts[a.ID]))
			}
		}
	}

	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}

	posts := &PostsResult{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		NextUrl:    "",
		PrevUrl:    "",
	}
	if items == nil {
		posts.Items = []any{}
	}

	extra := map[string]any{
		"posts":   posts,
		"keyword": keyword,
	}
	return r.renderPage(c, "index", "index", page, extra)
}

func articleIDs(articles []*models.Article) []int {
	ids := make([]int, len(articles))
	for i, a := range articles {
		ids[i] = a.ID
	}
	return ids
}

func (r *ThemeRenderer) renderTag(c fiber.Ctx, slug string, page int) error {
	ctx := c.Context()
	tag, err := r.tagService.GetBySlug(ctx, slug)
	if err != nil || tag == nil {
		return c.Status(fiber.StatusNotFound).SendString("Tag not found")
	}
	tagObj := map[string]any{
		"spec": map[string]any{
			"displayName": tag.Name,
			"slug":        tag.Slug,
		},
		"metadata": map[string]any{
			"name":        tag.Slug,
			"displayName": tag.Name,
		},
		"status": map[string]any{
			"permalink":   "/tags/" + tag.Slug,
			"displayName": tag.Name,
		},
	}

	if page < 1 {
		page = 1
	}
	pageSize := 10
	offset := (page - 1) * pageSize
	articles, total, err := r.articleService.PublicList(ctx, offset, pageSize, "", tag.Slug)

	commentCounts, _ := r.commentService.CountByArticles(ctx, articleIDs(articles))
	var items []any
	for _, a := range articles {
		tags, _ := r.articleService.GetTags(ctx, a.ID)
		var author *models.User
		if u, err := r.userService.GetByID(ctx, a.UserID); err == nil {
			author = u
		}
		categories := r.getPostCategories(a.Categories)
		items = append(items, buildHaloPost(a, tags, author, categories, commentCounts[a.ID]))
	}
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	posts := &PostsResult{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		NextUrl:    nextPageURL(page, totalPages),
		PrevUrl:    prevPageURL(page),
	}

	extra := map[string]any{
		"tag":   tagObj,
		"posts": posts,
	}
	return r.renderPage(c, "tag", "tag", page, extra)
}

// renderPage is the common rendering pipeline for all theme pages.
// templateName is the file name (without .html) under templates/.
// htmlType is the value exposed to templates as ${htmlType}.
// extra variables are added to the context after the common ones.
func (r *ThemeRenderer) renderPage(c fiber.Ctx, templateName, htmlType string, page int, extra map[string]any) error {
	ctx := c.Context()

	if err := r.ensureEngine(ctx); err != nil {
		return c.Status(fiber.StatusNotFound).SendString("No active theme: " + err.Error())
	}

	renderCtx, err := r.buildRenderContext(ctx, htmlType, page)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Failed to build render context: " + err.Error())
	}

	// Fallback site.url to the request base URL when no system setting is configured.
	// This ensures post copyright links and theme JS config get a real base URL.
	if siteVal, ok := renderCtx.Get("site"); ok {
		if site, ok := siteVal.(map[string]any); ok {
			if u, ok := site["url"].(string); !ok || u == "" || u == "/" {
				site["url"] = c.BaseURL() + "/"
			}
		}
	}

	for k, v := range extra {
		renderCtx.Set(k, v)
	}

	// Override title for post/category/tag pages
	if htmlType == "post" {
		if post, ok := extra["post"].(map[string]any); ok {
			if spec, ok := post["spec"].(map[string]any); ok {
				if title, ok := spec["title"].(string); ok {
					renderCtx.Set("title", title)
				}
			}
		}
	}

	r.mu.RLock()
	engine := r.engine
	r.mu.RUnlock()

	html, err := engine.Render(templateName, renderCtx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).SendString("Template render error: " + err.Error())
	}

	html = injectHaloWidgets(html)
	html = injectHaloFooter(html)
	html = injectHaloComment(html, extra)
	html = injectWordCountGuard(html)

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(html)
}

// buildArchivesObject groups articles by year and month for the archives template.
func (r *ThemeRenderer) buildArchivesObject(articles []*models.Article, total int, page, pageSize int, baseURL string) *ArchivesResult {
	type monthKey struct {
		year  int
		month int
	}
	grouped := map[monthKey][]*models.Article{}
	for _, a := range articles {
		y, m, _ := a.CreatedAt.Date()
		grouped[monthKey{y, int(m)}] = append(grouped[monthKey{y, int(m)}], a)
	}

	// Sort year/month keys descending
	var keys []monthKey
	for k := range grouped {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j].year > keys[i].year || (keys[j].year == keys[i].year && keys[j].month > keys[i].month) {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}

	var items []ArchiveYear
	for _, k := range keys {
		var monthPosts []any
		commentCounts, _ := r.commentService.CountByArticles(context.Background(), articleIDs(grouped[k]))
		for _, a := range grouped[k] {
			tags, _ := r.articleService.GetTags(context.Background(), a.ID)
			var author *models.User
			if u, err := r.userService.GetByID(context.Background(), a.UserID); err == nil {
				author = u
			}
			categories := r.getPostCategories(a.Categories)
		monthPosts = append(monthPosts, buildHaloPost(a, tags, author, categories, commentCounts[a.ID]))
		}
		items = append(items, ArchiveYear{
			Year: k.year,
			Months: []ArchiveMonth{
				{Month: k.month, Posts: monthPosts},
			},
		})
	}
	if items == nil {
		items = []ArchiveYear{}
	}

	totalPages := total / pageSize
	if total%pageSize > 0 {
		totalPages++
	}
	return &ArchivesResult{
		Items:      items,
		Total:      total,
		Page:       page,
		TotalPages: totalPages,
		PrevUrl:    prevPageURL(page),
		NextUrl:    nextPageURL(page, totalPages),
	}
}

// ArchivesResult groups posts by year/month for the archives template.
type ArchivesResult struct {
	Items      []ArchiveYear `json:"items"`
	Total      int           `json:"total"`
	Page       int           `json:"page"`
	PageSize   int           `json:"pageSize"`
	TotalPages int           `json:"totalPages"`
	PrevUrl    string        `json:"prevUrl"`
	NextUrl    string        `json:"nextUrl"`
}

func (a *ArchivesResult) HasNext() bool     { return a.Page < a.TotalPages }
func (a *ArchivesResult) HasPrevious() bool { return a.Page > 1 }

// ArchiveYear groups posts by year.
type ArchiveYear struct {
	Year   int            `json:"year"`
	Months []ArchiveMonth `json:"months"`
}

// ArchiveMonth groups posts by month.
type ArchiveMonth struct {
	Month int   `json:"month"`
	Posts []any `json:"posts"`
}

// parsePageQuery parses the ?page=N query parameter.
func parsePageQuery(v string) int {
	if v == "" {
		return 1
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return 1
	}
	return n
}

// parsePageParam parses the /page/:n path parameter.
func parsePageParam(v string) int {
	return parsePageQuery(v)
}

// ServeThemeAsset serves static assets (CSS, JS, images) from the active theme.
func (r *ThemeRenderer) ServeThemeAsset(c fiber.Ctx) error {
	themeID := c.Params("id")
	assetPath := c.Params("*")

	if assetPath == "" {
		return c.Status(fiber.StatusNotFound).SendString("Not found")
	}

	fullPath, contentType, err := r.themeService.ServeThemeAsset(themeID, assetPath)
	if err != nil {
		return c.Status(fiber.StatusNotFound).SendString("Asset not found")
	}

	c.Set("Content-Type", contentType)
	return c.SendFile(fullPath)
}

// buildRenderContext creates the thymeleaf Context with all variables needed by Halo themes.
func (r *ThemeRenderer) buildRenderContext(ctx context.Context, htmlType string, page int) (*thymeleaf.Context, error) {
	renderCtx := thymeleaf.NewContext()

	// Set the i18n message resolver
	r.mu.RLock()
	if r.messageResolver != nil {
		renderCtx.SetMessageResolver(r.messageResolver)
	}
	r.mu.RUnlock()

	// Get page from query param if available
	if page < 1 {
		page = 1
	}

	// Build site object
	site := r.buildSiteObject(ctx)
	renderCtx.Set("site", site)

	// Build theme object
	theme, err := r.buildThemeObject(ctx)
	if err != nil {
		return nil, err
	}
	renderCtx.Set("theme", theme)

	// Set the theme base path for @{...} URL expressions and #theme.assets().
	// Theme assets are served at /themes/<theme-id>/...
	activeTheme, _ := r.themeService.GetActiveTheme(ctx)
	if activeTheme != nil && activeTheme.ID != "" {
		renderCtx.Set("__theme_base_path", "/themes/"+activeTheme.ID)
	}

	// Build posts object
	posts := r.buildPostsObject(ctx, page)
	renderCtx.Set("posts", posts)

	// Set page type and title
	renderCtx.Set("htmlType", htmlType)
	renderCtx.Set("title", toStr(site["title"]))

	// Build finders
	postFinder := NewPostFinder(r.articleService, r.userService)
	categoryFinder := NewCategoryFinder(r.categoryService)
	tagFinder := NewTagFinder(r.tagService)
	menuFinder := NewMenuFinder(r.menuService)
	pluginFinder := NewPluginFinder()
	commentFinder := NewCommentFinder(r.commentService)
	themeFinder := NewThemeFinder(theme)

	renderCtx.Set("postFinder", postFinder)
	renderCtx.Set("categoryFinder", categoryFinder)
	renderCtx.Set("tagFinder", tagFinder)
	renderCtx.Set("menuFinder", menuFinder)
	renderCtx.Set("pluginFinder", pluginFinder)
	renderCtx.Set("commentFinder", commentFinder)
	renderCtx.Set("themeFinder", themeFinder)
	renderCtx.Set("siteStatsFinder", NewSiteStatsFinder(r.articleService, r.commentService, r.categoryService))
	renderCtx.Set("contributorFinder", &ContributorFinder{})
	renderCtx.Set("thumbnail", &ThumbnailGenerator{})

	// Halo comment system toggle (defaults to enabled when setting is absent).
	enabled, _ := r.settingService.Get(ctx, "enable_comments")
	renderCtx.Set("haloCommentEnabled", enabled != "false")

	return renderCtx, nil
}

// buildSiteObject creates the site context map from DB settings.
func (r *ThemeRenderer) buildSiteObject(ctx context.Context) map[string]any {
	site := make(map[string]any)

	title, _ := r.settingService.Get(ctx, "site_title")
	site["title"] = title

	url, _ := r.settingService.Get(ctx, "site_url")
	if url == "" {
		url, _ = r.settingService.Get(ctx, "site_base_url")
	}
	if url == "" {
		url = "/"
	}
	site["url"] = url

	subtitle, _ := r.settingService.Get(ctx, "site_subtitle")
	site["subtitle"] = subtitle

	// Logo — admin settings only has "site_icon", so prefer it and fall back to legacy "site_logo".
	logo, _ := r.settingService.Get(ctx, "site_icon")
	if logo == "" {
		logo, _ = r.settingService.Get(ctx, "site_logo")
	}
	site["logo"] = logo

	// Favicon — admin settings uses "site_icon", legacy themes may use "site_favicon".
	favicon, _ := r.settingService.Get(ctx, "site_icon")
	if favicon == "" {
		favicon, _ = r.settingService.Get(ctx, "site_favicon")
	}
	if favicon == "" {
		favicon = logo
	}
	site["favicon"] = favicon

	author, _ := r.settingService.Get(ctx, "site_author")
	site["author"] = author

	copyright, _ := r.settingService.Get(ctx, "site_copyright")
	site["copyright"] = copyright

	// SEO
	seo := make(map[string]any)
	desc, _ := r.settingService.Get(ctx, "site_description")
	seo["description"] = desc
	kw, _ := r.settingService.Get(ctx, "site_keywords")
	seo["keywords"] = kw
	site["seo"] = seo

	return site
}

// buildThemeObject creates the theme context map with spec and config.
func (r *ThemeRenderer) buildThemeObject(ctx context.Context) (map[string]any, error) {
	activeTheme, err := r.themeService.GetActiveTheme(ctx)
	if err != nil {
		return nil, err
	}

	theme := make(map[string]any)

	// Theme spec
	spec := make(map[string]any)
	spec["version"] = activeTheme.Version
	spec["displayName"] = activeTheme.Name
	spec["author"] = activeTheme.Author
	spec["repo"] = activeTheme.Repo
	spec["homepage"] = activeTheme.Homepage
	theme["spec"] = spec

	// Theme config (nested from flat DB keys)
	config := make(map[string]any)
	if activeTheme.SettingName != "" {
		settingsResp, err := r.themeService.GetThemeSettings(ctx, activeTheme.ID)
		if err == nil && settingsResp != nil {
			// Merge defaults with DB values
			var defaults map[string]any
			themeDir, _ := r.themeService.GetThemeDir(activeTheme.ID)
			if themeDir != "" {
				if def, err := r.themeService.ParseSettingsYAML(themeDir); err == nil {
					defaults = r.themeService.ExtractDefaults(def)
					for k, v := range defaults {
						config[k] = v
					}
				}
			}

			// Build reverse map from leaf field name to full dotted default key so that
			// DB values saved with unprefixed keys (e.g. "theme_mode") can be mapped to
			// their canonical group-prefixed path (e.g. "basic.theme_mode").
			// Sort keys first to ensure deterministic results.
			sortedKeys := make([]string, 0, len(defaults))
			for k := range defaults {
				sortedKeys = append(sortedKeys, k)
			}
			sort.Strings(sortedKeys)

			reverseDefaults := make(map[string]string, len(sortedKeys))
			for _, dottedKey := range sortedKeys {
				parts := strings.Split(dottedKey, ".")
				reverseDefaults[parts[len(parts)-1]] = dottedKey
			}

			// Override with actual DB values — must be deterministic.
			//
			// settingsResp.Values may contain BOTH dotted keys (e.g. "basic.theme_mode",
			// saved by Activate → ExtractDefaults) and unprefixed keys (e.g. "theme_mode",
			// saved by the admin panel). When an unprefixed key resolves to the same
			// qualified path as a dotted key, the result wins depending on Go map
			// iteration order, which is random.
			//
			// Two-pass approach: apply dotted (theme defaults) keys first,
			// then unprefixed (admin) keys so admin overrides always win and
			// the result is deterministic regardless of iteration order.
			for k, v := range settingsResp.Values {
				if strings.Contains(k, ".") {
					config[k] = v
				}
			}
			for k, v := range settingsResp.Values {
				if !strings.Contains(k, ".") {
					if dottedKey, ok := reverseDefaults[k]; ok {
						config[dottedKey] = v
					} else {
						config[k] = v
					}
				}
			}
		}
	}

	// Convert flat dotted keys to nested map
	theme["config"] = flattenToNested(config)

	return theme, nil
}

// buildPostsObject creates the paginated posts result.
func (r *ThemeRenderer) buildPostsObject(ctx context.Context, page int) *PostsResult {
	pageSize := 10
	offset := (page - 1) * pageSize

	articles, total, err := r.articleService.PublicList(ctx, offset, pageSize, "", "")
	if err != nil || len(articles) == 0 {
		return &PostsResult{
			Items:      []any{},
			Total:      0,
			Page:       page,
			PageSize:   pageSize,
			TotalPages: 0,
		}
	}

	commentCounts, _ := r.commentService.CountByArticles(ctx, articleIDs(articles))
	var items []any
	for _, a := range articles {
		tags, _ := r.articleService.GetTags(ctx, a.ID)
		var author *models.User
		if u, err := r.userService.GetByID(ctx, a.UserID); err == nil {
			author = u
		}

		categories := r.getPostCategories(a.Categories)
		post := buildHaloPost(a, tags, author, categories, commentCounts[a.ID])
		items = append(items, post)
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &PostsResult{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		NextUrl:    nextPageURL(page, totalPages),
		PrevUrl:    prevPageURL(page),
	}
}

// getPostCategories returns categories for a post.
func (r *ThemeRenderer) getPostCategories(categories []*models.Category) []map[string]any {
	if len(categories) == 0 {
		return []map[string]any{}
	}
	var result []map[string]any
	for _, cat := range categories {
		result = append(result, map[string]any{
			"spec": map[string]any{
				"displayName": cat.Name,
				"slug":        cat.Slug,
			},
			"metadata": map[string]any{
				"name":        cat.Slug,
				"displayName": cat.Name,
			},
			"status": map[string]any{
				"permalink":        "/categories/" + cat.Slug,
				"visiblePostCount": 0,
			},
			"icon": cat.Icon,
		})
	}
	return result
}

// ─── Data Types ───

// PostsResult wraps paginated posts for template access.
type PostsResult struct {
	Items      []any  `json:"items"`
	Total      int    `json:"total"`
	Page       int    `json:"page"`
	PageSize   int    `json:"pageSize"`
	TotalPages int    `json:"totalPages"`
	PrevUrl    string `json:"prevUrl"`
	NextUrl    string `json:"nextUrl"`
}

// HasNext returns true if there is a next page (template: ${posts.hasNext()}).
func (r *PostsResult) HasNext() bool { return r.Page < r.TotalPages }

// HasPrevious returns true if there is a previous page (template: ${posts.hasPrevious()}).
func (r *PostsResult) HasPrevious() bool { return r.Page > 1 }

// buildHaloPost converts an Article to a Halo-compatible post map.
func buildHaloPost(a *models.Article, tags []*models.Tag, author *models.User, categories []map[string]any, commentCount int) map[string]any {
	annotations := defaultPostAnnotations()
	if a.CommentEnabled {
		annotations["enable_comment"] = "true"
	} else {
		annotations["enable_comment"] = "false"
	}

	post := map[string]any{
		"spec": map[string]any{
			"title":       a.Title,
			"slug":        a.Slug,
			"excerpt":     a.Summary,
			"cover":       a.CoverURL,
			"releaseTime": a.CreatedAt.Format("2006-01-02T15:04:05Z"),
			"published":   a.Status == "published",
			"publishTime": a.CreatedAt.Format("2006-01-02T15:04:05Z"),
			"pinned":      false,
			"visible": map[string]any{
				"name": "PUBLIC",
			},
		},
		"stats": map[string]any{
			"visit":   a.ViewCount,
			"comment": commentCount,
			"upvote":  a.UpvoteCount,
		},
		"metadata": map[string]any{
			"name":        a.Slug,
			"displayName": a.Title,
			"annotations": annotations,
		},
		"content": map[string]any{
			"raw":     a.Content,
			"content": a.Content,
		},
		"status": map[string]any{
			"permalink": "/archives/" + a.Slug,
			"excerpt":   a.Summary,
		},
	}

	// Contributors are used by the copyright block (post_copyright.html).
	// Prefer nickname, fall back to username.
	if author != nil {
		name := author.Username
		if author.Nickname != "" {
			name = author.Nickname
		}
		post["status"].(map[string]any)["contributors"] = []string{name}
	} else {
		post["status"].(map[string]any)["contributors"] = []string{}
	}

	post["categories"] = categories

	if author != nil {
		post["owner"] = map[string]any{
			"name":        author.Username,
			"displayName": author.Nickname,
			"avatar":      author.Avatar,
			"bio":         author.Bio,
			"permalink":   "/u/" + author.Username,
		}
	}

	var tagList []map[string]any
	for _, t := range tags {
		tagList = append(tagList, map[string]any{
			"spec": map[string]any{
				"displayName": t.Name,
				"slug":        t.Slug,
				"cover":       t.Icon,
			},
			"metadata": map[string]any{
				"name":        t.Slug,
				"displayName": t.Name,
			},
			"status": map[string]any{
				"permalink":   "/tags/" + t.Slug,
				"displayName": t.Name,
			},
			"icon": t.Icon,
		})
	}
	if tagList == nil {
		tagList = []map[string]any{}
	}
	post["tags"] = tagList

	return post
}

// buildHaloTagList converts model tags to the Halo-style map list used by templates.
func buildHaloTagList(tags []*models.Tag, counts map[int]int) []map[string]any {
	var result []map[string]any
	for _, t := range tags {
		result = append(result, map[string]any{
			"spec": map[string]any{
				"displayName": t.Name,
				"slug":        t.Slug,
				"cover":       t.Icon,
			},
			"metadata": map[string]any{
				"name":        t.Slug,
				"displayName": t.Name,
			},
			"status": map[string]any{
				"permalink":        "/tags/" + t.Slug,
				"displayName":      t.Name,
				"visiblePostCount": counts[t.ID],
			},
			"icon": t.Icon,
		})
	}
	if result == nil {
		result = []map[string]any{}
	}
	return result
}

// ContributorFinder provides contributor info for templates.
type ContributorFinder struct{}

func (f *ContributorFinder) GetContributor(name string) map[string]any {
	return map[string]any{
		"name":        name,
		"displayName": name,
		"avatar":      "",
		"bio":         "",
	}
}

// ThumbnailGenerator provides thumbnail URL generation for templates.
type ThumbnailGenerator struct{}

func (g *ThumbnailGenerator) Gen(url string, size string) string {
	return url
}

// ─── Helpers ───

// flattenToNested converts flat dotted keys to nested maps.
func flattenToNested(flat map[string]any) map[string]any {
	result := make(map[string]any)
	for k, v := range flat {
		parts := strings.Split(k, ".")
		current := result
		for i, part := range parts {
			if i == len(parts)-1 {
				current[part] = v
			} else {
				if next, ok := current[part].(map[string]any); ok {
					current = next
				} else {
					next = make(map[string]any)
					current[part] = next
					current = next
				}
			}
		}
	}
	return result
}

// injectHaloFooter removes the <halo:footer> placeholder that Halo normally
// replaces with plugin widget scripts. The real SearchWidget implementation is
// provided by injectHaloWidgets; this function just cleans up the placeholder
// so it doesn't render as stray text.
func injectHaloFooter(html string) string {
	if !strings.Contains(html, "<halo:footer") {
		return html
	}

	// Self-closing tags
	html = strings.ReplaceAll(html, "<halo:footer />", "")
	html = strings.ReplaceAll(html, "<halo:footer/>", "")

	// Paired tags (possibly with whitespace/content)
	footerRe := regexp.MustCompile(`(?i)<halo:footer\b[^>]*>[\s\S]*?</halo:footer>`)
	html = footerRe.ReplaceAllString(html, "")

	return html
}

// injectHaloComment converts the Thymeleaf custom tag <halo:comment> into a
// runtime switcher. Anonymous visitors see the giscus comment system; logged-in
// users (auth_token in localStorage) see the built-in Halo comment widget so
// they can comment with their site identity.
func injectHaloComment(html string, extra map[string]any) string {
	if !strings.Contains(html, "<halo:comment") {
		return html
	}

	allowComment := isCommentAllowed(extra)
	allowAttr := "false"
	if allowComment {
		allowAttr = "true"
	}
	defaultName, defaultKind := commentSubject(extra)

	// Paired tags first: the Thymeleaf engine sometimes expands self-closing
	// custom tags into <halo:comment>...</halo:comment>, so this must run
	// before the self-closing regex to avoid leaving a stray closing tag.
	pairedRe := regexp.MustCompile(`(?i)<halo:comment\b([^>]*)>([\s\S]*?)</halo:comment>`)
	html = pairedRe.ReplaceAllStringFunc(html, func(match string) string {
		groups := pairedRe.FindStringSubmatch(match)
		attrs, inner := groups[1], groups[2]
		attrs = stripDataAllowComment(attrs)
		attrs = ensureCommentAttrs(attrs, extra)
		return renderCommentSwitcher(attrs, inner, defaultName, defaultKind, allowAttr)
	})

	// Remaining self-closing tags.
	re := regexp.MustCompile(`(?i)<halo:comment\b([^>]*)/>`)
	html = re.ReplaceAllStringFunc(html, func(match string) string {
		attrs := re.FindStringSubmatch(match)[1]
		attrs = stripDataAllowComment(attrs)
		attrs = ensureCommentAttrs(attrs, extra)
		return renderCommentSwitcher(attrs, "", defaultName, defaultKind, allowAttr)
	})

	return html
}

// renderCommentSwitcher emits a placeholder and an inline script that decides
// whether to load the Halo comment widget (logged-in) or giscus (anonymous).
func renderCommentSwitcher(attrs, inner, defaultName, defaultKind, allowComment string) string {
	name := extractAttr(attrs, "name")
	if name == "" {
		name = defaultName
	}
	kind := extractAttr(attrs, "kind")
	if kind == "" {
		kind = defaultKind
	}
	if kind == "" {
		kind = "Post"
	}

	return `<div class="halo-comment-switcher" data-name="` + html.EscapeString(name) + `" data-kind="` + html.EscapeString(kind) + `" data-allow-comment="` + allowComment + `">` + inner + `</div>` +
		`<script>
(function(){
  var token = localStorage.getItem('auth_token');
  var container = document.currentScript.previousElementSibling;
  if (!container) return;
  if (token) {
    container.innerHTML = '<halo-comment name="` + name + `" kind="` + kind + `" data-allow-comment="` + allowComment + `"></halo-comment>';
  } else {
    container.innerHTML = '<div class="giscus"></div>';
    var s = document.createElement('script');
    s.src = 'https://giscus.app/client.js';
    s.setAttribute('data-repo','eefenaxce/giscus');
    s.setAttribute('data-repo-id','R_kgDOThr-HQ');
    s.setAttribute('data-category','Announcements');
    s.setAttribute('data-category-id','DIC_kwDOThr-Hc4DB1MR');
    s.setAttribute('data-mapping','pathname');
    s.setAttribute('data-strict','0');
    s.setAttribute('data-reactions-enabled','1');
    s.setAttribute('data-emit-metadata','0');
    s.setAttribute('data-input-position','top');
    s.setAttribute('data-theme','preferred_color_scheme');
    s.setAttribute('data-lang','zh-CN');
    s.setAttribute('data-loading','lazy');
    s.setAttribute('crossorigin','anonymous');
    s.async = true;
    container.appendChild(s);
  }
})();
</script>`
}

// jsStringLiteral returns a single-quoted JavaScript string literal for s.
func jsStringLiteral(s string) string {
	var b strings.Builder
	b.WriteByte('\'')
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString("\\\\")
		case '\'':
			b.WriteString("\\'")
		case '\n':
			b.WriteString("\\n")
		case '\r':
			b.WriteString("\\r")
		case '\t':
			b.WriteString("\\t")
		case '<':
			b.WriteString("\\u003c")
		case '>':
			b.WriteString("\\u003e")
		case '&':
			b.WriteString("\\u0026")
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('\'')
	return b.String()
}

// extractAttr returns the value of a quoted HTML attribute from attrs.
func extractAttr(attrs, name string) string {
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(name) + `=["']([^"']*)["']`)
	if m := re.FindStringSubmatch(attrs); m != nil {
		return m[1]
	}
	return ""
}

// commentSubject extracts the Halo-style subject name and kind for the current
// page. Halo themes pass these via fragment parameters, but if the engine
// doesn't render them on <halo:comment> we fall back to the post/page context.
func commentSubject(extra map[string]any) (name, kind string) {
	if extra == nil {
		return "", "Post"
	}
	for _, key := range []string{"post", "singlePage"} {
		obj, ok := extra[key].(map[string]any)
		if !ok {
			continue
		}
		if key == "singlePage" {
			kind = "SinglePage"
		} else {
			kind = "Post"
		}
		if metadata, ok := obj["metadata"].(map[string]any); ok {
			if v, ok := metadata["name"].(string); ok && v != "" {
				name = v
			}
		}
		if name == "" {
			if spec, ok := obj["spec"].(map[string]any); ok {
				if v, ok := spec["slug"].(string); ok && v != "" {
					name = v
				}
			}
		}
		if name != "" {
			return name, kind
		}
	}
	return "", "Post"
}

// hasAttr reports whether the given attribute string already contains attr.
func hasAttr(attrs, attr string) bool {
	re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(attr) + `\b`)
	return re.MatchString(attrs)
}

// ensureCommentAttrs adds name/kind attributes to a <halo-comment> attribute
// string when they are missing, using the current page context.
func ensureCommentAttrs(attrs string, extra map[string]any) string {
	name, kind := commentSubject(extra)
	if name != "" && !hasAttr(attrs, "name") {
		attrs += ` name="` + name + `"`
	}
	if !hasAttr(attrs, "kind") {
		attrs += ` kind="` + kind + `"`
	}
	return attrs
}

// isCommentAllowed checks the current post's metadata.annotations.enable_comment
// to decide whether the halo-comment widget should allow comments.
func isCommentAllowed(extra map[string]any) bool {
	if extra == nil {
		return true
	}
	post, ok := extra["post"].(map[string]any)
	if !ok {
		return true
	}
	metadata, ok := post["metadata"].(map[string]any)
	if !ok {
		return true
	}
	annotations, ok := metadata["annotations"].(map[string]any)
	if !ok {
		return true
	}
	v, ok := annotations["enable_comment"]
	if !ok {
		return true
	}
	return toStr(v) != "false"
}

// stripDataAllowComment removes any existing data-allow-comment attribute from
// the tag attribute string so we don't duplicate it.
func stripDataAllowComment(attrs string) string {
	re := regexp.MustCompile(`(?i)\s+data-allow-comment="[^"]*"`)
	attrs = re.ReplaceAllString(attrs, "")
	re = regexp.MustCompile(`(?i)\s+data-allow-comment='[^']*'`)
	attrs = re.ReplaceAllString(attrs, "")
	re = regexp.MustCompile(`(?i)\s+data-allow-comment`)
	attrs = re.ReplaceAllString(attrs, "")
	return attrs
}

// injectWordCountGuard prevents "Cannot set properties of null (setting 'innerText')"
// when the word-count element is not rendered (e.g. enable_page_meta is disabled).
// The theme's post.html/page.html scripts unconditionally update #wordCount innerText,
// so we patch the rendered HTML to guard the element lookup.
func injectWordCountGuard(html string) string {
	old := "document.getElementById('wordCount').innerText = `${wordCount} 字`;"
	new := "const _wcEl = document.getElementById('wordCount'); if (_wcEl) _wcEl.innerText = `${wordCount} 字`;"
	return strings.ReplaceAll(html, old, new)
}

func toStr(v any) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(val)
	case int64:
		return strconv.FormatInt(val, 10)
	}
	return ""
}
