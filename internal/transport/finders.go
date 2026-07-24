package transport

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/eefenaxce/axce_blog/internal/models"
	"github.com/eefenaxce/axce_blog/internal/service"
)

// ─── Halo 数据模型包装器 ───
// Halo 主题模板使用嵌套结构: ${post.spec.title}, ${post.stats.visit} 等。
// 这些包装器将现有的扁平 model 映射到 Halo 兼容的结构。

// HaloMetadata 模拟 Halo 的 metadata 对象
type HaloMetadata struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"displayName"`
	Annotations map[string]any `json:"annotations"`
}

// HaloVisible 模拟 Halo 的可见性对象
type HaloVisible struct {
	Name string `json:"name"`
}

// HaloPostSpec 模拟 Halo Post 的 spec 对象
type HaloPostSpec struct {
	Title       string       `json:"title"`
	Slug        string       `json:"slug"`
	Excerpt     string       `json:"excerpt"`
	Cover       string       `json:"cover"`
	ReleaseTime string       `json:"releaseTime"`
	Deleted     bool         `json:"deleted"`
	Published   bool         `json:"published"`
	PublishTime string       `json:"publishTime"`
	Visible     *HaloVisible `json:"visible"`
}

// HaloPostStats 模拟 Halo Post 的 stats 对象
type HaloPostStats struct {
	Visit   int64 `json:"visit"`
	Comment int64 `json:"comment"`
	Upvote  int64 `json:"upvote"`
}

// HaloPostContent 模拟 Halo Post 的 content 对象
type HaloPostContent struct {
	Raw     string `json:"raw"`
	Content string `json:"content"`
}

// HaloPost 包装 Article 为 Halo 兼容的 Post 结构
type HaloPost struct {
	Spec       HaloPostSpec    `json:"spec"`
	Stats      HaloPostStats   `json:"stats"`
	Metadata   HaloMetadata    `json:"metadata"`
	Content    HaloPostContent `json:"content"`
	Categories []*HaloCategory `json:"categories"`
	Tags       []*HaloTag      `json:"tags"`
	Owner      *HaloAuthor     `json:"owner"`
	Status     HaloPostStatus  `json:"status"`
}

// HaloPostStatus holds permalink, excerpt, contributors for templates.
type HaloPostStatus struct {
	Permalink    string   `json:"permalink"`
	Excerpt      string   `json:"excerpt"`
	Contributors []string `json:"contributors"`
}

// HaloCategorySpec 模拟 Halo Category 的 spec
type HaloCategorySpec struct {
	DisplayName string `json:"displayName"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
}

// HaloCategory 包装 Category 为 Halo 兼容结构
type HaloCategory struct {
	Spec      HaloCategorySpec   `json:"spec"`
	Metadata  HaloMetadata       `json:"metadata"`
	Status    HaloCategoryStatus `json:"status"`
	PostCount int64              `json:"postCount"`
	Icon      string             `json:"icon"`
	Children  []*HaloCategory    `json:"children"`
}

// HaloCategoryStatus holds permalink and post count for templates.
type HaloCategoryStatus struct {
	Permalink        string `json:"permalink"`
	DisplayName      string `json:"displayName"`
	VisiblePostCount int64  `json:"visiblePostCount"`
}

// HaloTagSpec 模拟 Halo Tag 的 spec
type HaloTagSpec struct {
	DisplayName string `json:"displayName"`
	Slug        string `json:"slug"`
	Cover       string `json:"cover"`
}

// HaloTag 包装 Tag 为 Halo 兼容结构
type HaloTag struct {
	Spec      HaloTagSpec   `json:"spec"`
	Metadata  HaloMetadata  `json:"metadata"`
	Status    HaloTagStatus `json:"status"`
	PostCount int64         `json:"postCount"`
	Icon      string        `json:"icon"`
}

// HaloTagStatus holds permalink and post count for templates.
type HaloTagStatus struct {
	Permalink        string `json:"permalink"`
	DisplayName      string `json:"displayName"`
	VisiblePostCount int    `json:"visiblePostCount"`
}

// HaloAuthor 模拟 Halo 作者对象
type HaloAuthor struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Avatar      string `json:"avatar"`
	Bio         string `json:"bio"`
	Permalink   string `json:"permalink"`
}

// HaloMenuItemTarget mirrors Halo's target spec for menu items.
type HaloMenuItemTarget struct {
	Value string `json:"value"`
}

// HaloMenuItemSpec mirrors Halo's menu item spec.
type HaloMenuItemSpec struct {
	Target *HaloMenuItemTarget `json:"target"`
}

// HaloMenuItemStatus mirrors Halo's menu item status.
type HaloMenuItemStatus struct {
	Href        string `json:"href"`
	DisplayName string `json:"displayName"`
}

// HaloMenuItem 模拟 Halo 菜单项
type HaloMenuItem struct {
	Spec     HaloMenuItemSpec   `json:"spec"`
	Status   HaloMenuItemStatus `json:"status"`
	Children []*HaloMenuItem    `json:"children"`
}

// HaloListResult 分页列表结果（Halo 风格）
type HaloListResult struct {
	Items      []*HaloPost `json:"items"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"pageSize"`
	TotalPages int         `json:"totalPages"`
	PrevUrl    string      `json:"prevUrl"`
	NextUrl    string      `json:"nextUrl"`
}

// HasNext returns true if there is a next page (template: ${posts.hasNext()}).
func (r *HaloListResult) HasNext() bool {
	return r.Page < r.TotalPages
}

// HasPrevious returns true if there is a previous page (template: ${posts.hasPrevious()}).
func (r *HaloListResult) HasPrevious() bool {
	return r.Page > 1
}

// ─── 转换函数 ───

func articleToHaloPost(a *models.Article, tags []*models.Tag, author *models.User) *HaloPost {
	visibleName := "PUBLIC"
	if a.Status != "published" {
		visibleName = "PRIVATE"
	}
	annotations := defaultPostAnnotations()
	if a.CommentEnabled {
		annotations["enable_comment"] = "true"
	} else {
		annotations["enable_comment"] = "false"
	}
	p := &HaloPost{
		Spec: HaloPostSpec{
			Title:       a.Title,
			Slug:        a.Slug,
			Excerpt:     a.Summary,
			Cover:       a.CoverURL,
			ReleaseTime: a.CreatedAt.Format("2006-01-02T15:04:05Z"),
			Published:   a.Status == "published",
			PublishTime: a.CreatedAt.Format("2006-01-02T15:04:05Z"),
			Visible:     &HaloVisible{Name: visibleName},
		},
		Stats: HaloPostStats{
			Visit:   int64(a.ViewCount),
			Comment: 0,
		},
		Metadata: HaloMetadata{
			Name:        a.Slug,
			DisplayName: a.Title,
			Annotations: annotations,
		},
		Content: HaloPostContent{
			Raw:     a.Content,
			Content: a.Content,
		},
		Status: HaloPostStatus{
			Permalink:    "/archives/" + a.Slug,
			Excerpt:      a.Summary,
			Contributors: []string{},
		},
	}
	if author != nil {
		p.Owner = &HaloAuthor{
			Name:        author.Username,
			DisplayName: author.Nickname,
			Avatar:      author.Avatar,
			Bio:         author.Bio,
			Permalink:   "/u/" + author.Username,
		}
	}
	for _, t := range tags {
		p.Tags = append(p.Tags, tagToHaloTag(t, nil))
	}
	return p
}

func categoryToHaloCategory(c *models.Category, count int64) *HaloCategory {
	return &HaloCategory{
		Spec: HaloCategorySpec{
			DisplayName: c.Name,
			Slug:        c.Slug,
			Description: c.Description,
		},
		Metadata: HaloMetadata{
			Name:        c.Slug,
			DisplayName: c.Name,
		},
		Status: HaloCategoryStatus{
			Permalink:        "/categories/" + c.Slug,
			DisplayName:      c.Name,
			VisiblePostCount: count,
		},
		PostCount: count,
		Icon:      c.Icon,
	}
}

func tagToHaloTag(t *models.Tag, counts map[int]int) *HaloTag {
	count := counts[t.ID]
	return &HaloTag{
		Spec: HaloTagSpec{
			DisplayName: t.Name,
			Slug:        t.Slug,
			Cover:       t.Icon,
		},
		Metadata: HaloMetadata{
			Name:        t.Slug,
			DisplayName: t.Name,
		},
		Status: HaloTagStatus{
			Permalink:        "/tags/" + t.Slug,
			DisplayName:      t.Name,
			VisiblePostCount: count,
		},
		PostCount: int64(count),
		Icon:      t.Icon,
	}
}

// ─── Finder 对象 ───
// 主题模板调用: ${postFinder.list(1,10)}, ${categoryFinder.listAsTree()} 等。

// PostFinder 文章查找器
type PostFinder struct {
	articleService *service.ArticleService
	userService    *service.UserService
}

func NewPostFinder(articleSvc *service.ArticleService, userSvc *service.UserService) *PostFinder {
	return &PostFinder{
		articleService: articleSvc,
		userService:    userSvc,
	}
}

// List 返回分页文章列表（Halo 风格: postFinder.list(page, size) 或 postFinder.list({page: 1, size: 10})）
func (f *PostFinder) List(params ...any) *HaloListResult {
	page, size := 1, 10
	if len(params) == 1 {
		if m, ok := params[0].(map[string]any); ok {
			page = toInt(m["page"])
			size = toInt(m["size"])
		} else {
			page = toInt(params[0])
		}
	} else if len(params) >= 2 {
		page = toInt(params[0])
		size = toInt(params[1])
	}
	return f.listBy(page, size, "", "")
}

// ListByCategory returns posts in a given category (template: postFinder.listByCategory(page, size, name)).
func (f *PostFinder) ListByCategory(page, size int, categorySlug any) *HaloListResult {
	return f.listBy(page, size, toStr(categorySlug), "")
}

// ListByTag returns posts with a given tag (template: postFinder.listByTag(page, size, name)).
func (f *PostFinder) ListByTag(page, size int, tagSlug any) *HaloListResult {
	return f.listBy(page, size, "", toStr(tagSlug))
}

// Search returns posts matching a keyword (template: postFinder.search(keyword, page, size)).
func (f *PostFinder) Search(keyword any, params ...any) *HaloListResult {
	q := toStr(keyword)
	page, size := 1, 10
	if len(params) >= 1 {
		page = toInt(params[0])
	}
	if len(params) >= 2 {
		size = toInt(params[1])
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	offset := (page - 1) * size

	articles, total, err := f.articleService.Search(context.Background(), q, offset, size)
	if err != nil || len(articles) == 0 {
		return &HaloListResult{Items: []*HaloPost{}, Total: 0, Page: page, PageSize: size, TotalPages: 0}
	}

	var items []*HaloPost
	for _, a := range articles {
		tags, _ := f.articleService.GetTags(context.Background(), a.ID)
		var author *models.User
		if u, err := f.userService.GetByID(context.Background(), a.UserID); err == nil {
			author = u
		}
		items = append(items, articleToHaloPost(a, tags, author))
	}

	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}

	return &HaloListResult{
		Items:      items,
		Total:      int64(total),
		Page:       page,
		PageSize:   size,
		TotalPages: totalPages,
		NextUrl:    nextPageURL(page, totalPages),
		PrevUrl:    prevPageURL(page),
	}
}

func (f *PostFinder) listBy(page, size int, categorySlug, tagSlug string) *HaloListResult {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	offset := (page - 1) * size

	articles, total, err := f.articleService.PublicList(context.Background(), offset, size, categorySlug, tagSlug)
	if err != nil || len(articles) == 0 {
		return &HaloListResult{Items: []*HaloPost{}, Total: 0, Page: page, PageSize: size, TotalPages: 0}
	}

	var items []*HaloPost
	for _, a := range articles {
		tags, _ := f.articleService.GetTags(context.Background(), a.ID)
		var author *models.User
		if u, err := f.userService.GetByID(context.Background(), a.UserID); err == nil {
			author = u
		}
		items = append(items, articleToHaloPost(a, tags, author))
	}

	totalPages := int(total) / size
	if int(total)%size > 0 {
		totalPages++
	}

	return &HaloListResult{
		Items:      items,
		Total:      int64(total),
		Page:       page,
		PageSize:   size,
		TotalPages: totalPages,
		NextUrl:    nextPageURL(page, totalPages),
		PrevUrl:    prevPageURL(page),
	}
}

// Archives groups published posts by year/month for the timeline view
// (template: ${postFinder.archives(page, size)}).
func (f *PostFinder) Archives(page, size int) *ArchivesResult {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 9999
	}
	offset := (page - 1) * size

	articles, total, err := f.articleService.PublicList(context.Background(), offset, size, "", "")
	if err != nil || len(articles) == 0 {
		return &ArchivesResult{Items: []ArchiveYear{}, Total: 0, Page: page, PageSize: size, TotalPages: 0}
	}

	type monthKey struct {
		year  int
		month int
	}
	grouped := map[monthKey][]*models.Article{}
	for _, a := range articles {
		y, m, _ := a.CreatedAt.Date()
		grouped[monthKey{y, int(m)}] = append(grouped[monthKey{y, int(m)}], a)
	}

	var keys []monthKey
	for k := range grouped {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].year != keys[j].year {
			return keys[i].year > keys[j].year
		}
		return keys[i].month > keys[j].month
	})

	var items []ArchiveYear
	for _, k := range keys {
		var posts []any
		for _, a := range grouped[k] {
			tags, _ := f.articleService.GetTags(context.Background(), a.ID)
			var author *models.User
			if u, err := f.userService.GetByID(context.Background(), a.UserID); err == nil {
				author = u
			}
			posts = append(posts, articleToHaloPost(a, tags, author))
		}
		items = append(items, ArchiveYear{
			Year: k.year,
			Months: []ArchiveMonth{
				{Month: k.month, Posts: posts},
			},
		})
	}
	if items == nil {
		items = []ArchiveYear{}
	}

	totalPages := total / size
	if total%size > 0 {
		totalPages++
	}

	return &ArchivesResult{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   size,
		TotalPages: totalPages,
		PrevUrl:    prevPageURL(page),
		NextUrl:    nextPageURL(page, totalPages),
	}
}

// Cursor returns previous/next post navigation (template: postFinder.cursor(name)).
func (f *PostFinder) Cursor(name any) *PostCursor {
	slug := toStr(name)
	if slug == "" {
		return &PostCursor{}
	}
	current, err := f.articleService.GetBySlug(context.Background(), slug)
	if err != nil || current == nil {
		return &PostCursor{}
	}

	// Find all published articles ordered by created_at desc to find prev/next
	all, _, err := f.articleService.PublicList(context.Background(), 0, 1000, "", "")
	if err != nil {
		return &PostCursor{}
	}

	var prevArticle, nextArticle *models.Article
	for i, a := range all {
		if a.ID == current.ID {
			if i > 0 {
				prevArticle = all[i-1] // older post
			}
			if i < len(all)-1 {
				nextArticle = all[i+1] // newer post
			}
			break
		}
	}

	cursor := &PostCursor{}
	if prevArticle != nil {
		tags, _ := f.articleService.GetTags(context.Background(), prevArticle.ID)
		var author *models.User
		if u, err := f.userService.GetByID(context.Background(), prevArticle.UserID); err == nil {
			author = u
		}
		cursor.Previous = articleToHaloPost(prevArticle, tags, author)
	}
	if nextArticle != nil {
		tags, _ := f.articleService.GetTags(context.Background(), nextArticle.ID)
		var author *models.User
		if u, err := f.userService.GetByID(context.Background(), nextArticle.UserID); err == nil {
			author = u
		}
		cursor.Next = articleToHaloPost(nextArticle, tags, author)
	}
	return cursor
}

// PostCursor represents previous/next post navigation.
type PostCursor struct {
	Previous *HaloPost `json:"previous"`
	Next     *HaloPost `json:"next"`
}

func (c *PostCursor) HasPrevious() bool { return c.Previous != nil }
func (c *PostCursor) HasNext() bool     { return c.Next != nil }

// defaultPostAnnotations returns Halo-style metadata annotations used by themes
// to toggle per-post features such as comments, likes, and share.
func defaultPostAnnotations() map[string]any {
	return map[string]any{
		"enable_comment":       "true",
		"enable_page_meta":     "true",
		"enable_collect_check": "true",
		"enable_passage_tips":  "true",
		"enable_like":          "true",
		"enable_share":         "true",
		"enable_donate":        "true",
		"enable_read_limit":    "false",
		"use_raw_content":      "false",
		"img_align":            "center",
		"enable_aside":         "true",
	}
}

func nextPageURL(page, totalPages int) string {
	if page < totalPages {
		return "page/" + intToStr(page+1)
	}
	return ""
}

func prevPageURL(page int) string {
	if page > 1 {
		return "page/" + intToStr(page-1)
	}
	return ""
}

func intToStr(i int) string {
	return fmt.Sprintf("%d", i)
}

// Get 按 slug 获取单篇文章
func (f *PostFinder) Get(slug string) *HaloPost {
	a, err := f.articleService.GetBySlug(context.Background(), slug)
	if err != nil || a == nil {
		return nil
	}
	tags, _ := f.articleService.GetTags(context.Background(), a.ID)
	var author *models.User
	if u, err := f.userService.GetByID(context.Background(), a.UserID); err == nil {
		author = u
	}
	return articleToHaloPost(a, tags, author)
}

// CategoryFinder 分类查找器
type CategoryFinder struct {
	categoryService *service.CategoryService
}

func NewCategoryFinder(catSvc *service.CategoryService) *CategoryFinder {
	return &CategoryFinder{categoryService: catSvc}
}

// ListAll returns all categories (template: categoryFinder.listAll()).
func (f *CategoryFinder) ListAll() []*HaloCategory {
	cats, err := f.categoryService.List(context.Background())
	if err != nil {
		return []*HaloCategory{}
	}
	var result []*HaloCategory
	for _, c := range cats {
		result = append(result, categoryToHaloCategory(c.Category, int64(c.ArticleCount)))
	}
	return result
}

// List 返回分类列表（支持 categoryFinder.list()、list(1,10)、list({page:1, size:10})）
func (f *CategoryFinder) List(params ...any) []*HaloCategory {
	all := f.ListAll()
	page, size := 1, 0
	if len(params) == 1 {
		if m, ok := params[0].(map[string]any); ok {
			page = toInt(m["page"])
			size = toInt(m["size"])
		} else {
			page = toInt(params[0])
		}
	} else if len(params) >= 2 {
		page = toInt(params[0])
		size = toInt(params[1])
	}
	if size <= 0 {
		return all
	}
	if page < 1 {
		page = 1
	}
	start := (page - 1) * size
	if start >= len(all) {
		return []*HaloCategory{}
	}
	end := start + size
	if end > len(all) {
		end = len(all)
	}
	return all[start:end]
}

// ListAsTree 返回分类树（当前为扁平列表）
func (f *CategoryFinder) ListAsTree() []*HaloCategory {
	return f.List()
}

// TagFinder 标签查找器
type TagFinder struct {
	tagService *service.TagService
}

func NewTagFinder(tagSvc *service.TagService) *TagFinder {
	return &TagFinder{tagService: tagSvc}
}

// List 返回标签列表（支持 tagFinder.list()、list(1,10)、list({page:1, size:10})）
func (f *TagFinder) List(params ...any) []*HaloTag {
	all := f.ListAll()
	page, size := 1, 0
	if len(params) == 1 {
		if m, ok := params[0].(map[string]any); ok {
			page = toInt(m["page"])
			size = toInt(m["size"])
		} else {
			page = toInt(params[0])
		}
	} else if len(params) >= 2 {
		page = toInt(params[0])
		size = toInt(params[1])
	}
	if size <= 0 {
		return all
	}
	if page < 1 {
		page = 1
	}
	start := (page - 1) * size
	if start >= len(all) {
		return []*HaloTag{}
	}
	end := start + size
	if end > len(all) {
		end = len(all)
	}
	return all[start:end]
}

// ListAll returns all tags (template: tagFinder.listAll()).
func (f *TagFinder) ListAll() []*HaloTag {
	tags, err := f.tagService.List(context.Background())
	if err != nil {
		return []*HaloTag{}
	}
	counts, _ := f.tagService.GetTagPostCounts(context.Background())
	var result []*HaloTag
	for _, t := range tags {
		result = append(result, tagToHaloTag(t, counts))
	}
	return result
}

// GetByName returns a single tag by slug (template: tagFinder.getByName(name)).
func (f *TagFinder) GetByName(name any) *HaloTag {
	slug := toStr(name)
	if slug == "" {
		return nil
	}
	tags, err := f.tagService.List(context.Background())
	if err != nil {
		return nil
	}
	counts, _ := f.tagService.GetTagPostCounts(context.Background())
	for _, t := range tags {
		if t.Slug == slug {
			return tagToHaloTag(t, counts)
		}
	}
	return nil
}

// GetByName on CategoryFinder returns a category by slug.
func (f *CategoryFinder) GetByName(name any) *HaloCategory {
	slug := toStr(name)
	if slug == "" {
		return nil
	}
	cats, err := f.categoryService.List(context.Background())
	if err != nil {
		return nil
	}
	for _, c := range cats {
		if c.Category.Slug == slug {
			return categoryToHaloCategory(c.Category, int64(c.ArticleCount))
		}
	}
	return nil
}

// PluginFinder checks plugin availability (template: pluginFinder.available('PluginSearchWidget')).
type PluginFinder struct{}

func NewPluginFinder() *PluginFinder { return &PluginFinder{} }

// Available returns true for built-in system plugins that the theme expects
// (CommentWidgetPlugin and SearchWidget). All other plugins are considered unavailable.
func (f *PluginFinder) Available(name string) bool {
	switch name {
	case "PluginCommentWidget", "PluginSearchWidget", "CommentWidgetPlugin", "WalinePlugin":
		return true
	}
	return false
}

// ThemeFinder returns the active theme (template: themeFinder.activation()).
// Halo themes call this to shadow the global `theme` context variable with the
// same object — typically used inside th:with to access spec.displayName,
// spec.version, spec.repo for the "Powered by" footer block.
type ThemeFinder struct {
	theme map[string]any
}

func NewThemeFinder(theme map[string]any) *ThemeFinder {
	return &ThemeFinder{theme: theme}
}

// Activation returns the currently active theme object.
func (f *ThemeFinder) Activation() map[string]any {
	return f.theme
}

// MenuFinder 菜单查找器
type MenuFinder struct {
	menuService *service.MenuService
}

func NewMenuFinder(menuService *service.MenuService) *MenuFinder {
	return &MenuFinder{menuService: menuService}
}

func (f *MenuFinder) List() []*HaloMenuItem {
	return []*HaloMenuItem{}
}

// GetPrimary returns the primary menu.
func (f *MenuFinder) GetPrimary() map[string]any {
	return f.GetByName("primary")
}

// GetByName returns a menu by name with nested children.
func (f *MenuFinder) GetByName(name string) map[string]any {
	_, items, err := f.menuService.GetByName(context.Background(), name)
	if err != nil {
		return map[string]any{
			"menuItems": []any{},
			"metadata": map[string]any{
				"name":        name,
				"displayName": name,
			},
		}
	}

	// Build id -> item lookup and attach children
	itemMap := make(map[int]*HaloMenuItem)
	var roots []*HaloMenuItem
	for _, it := range items {
		itemMap[it.ID] = &HaloMenuItem{
			Spec: HaloMenuItemSpec{
				Target: &HaloMenuItemTarget{Value: "_self"},
			},
			Status: HaloMenuItemStatus{
				Href:        it.URL,
				DisplayName: it.Name,
			},
			Children: []*HaloMenuItem{},
		}
	}
	for _, it := range items {
		node := itemMap[it.ID]
		if it.ParentID != nil {
			if parent, ok := itemMap[*it.ParentID]; ok {
				parent.Children = append(parent.Children, node)
			} else {
				roots = append(roots, node)
			}
		} else {
			roots = append(roots, node)
		}
	}

	var menuItems []any
	for _, r := range roots {
		menuItems = append(menuItems, r)
	}
	return map[string]any{
		"menuItems": menuItems,
		"metadata": map[string]any{
			"name":        name,
			"displayName": name,
		},
	}
}

// CommentFinder provides Halo-style comments for templates.
type CommentFinder struct {
	commentService *service.CommentService
}

func NewCommentFinder(commentSvc *service.CommentService) *CommentFinder {
	return &CommentFinder{commentService: commentSvc}
}

// List returns recent approved comments (template: commentFinder.list(null, page, size)).
func (f *CommentFinder) List(params ...any) []map[string]any {
	page, size := 1, 10
	if len(params) >= 2 {
		page = toInt(params[1])
	}
	if len(params) >= 3 {
		size = toInt(params[2])
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	offset := (page - 1) * size

	if f.commentService == nil {
		return []map[string]any{}
	}
	comments, _, err := f.commentService.List(context.Background(), offset, size, "approved")
	if err != nil {
		return []map[string]any{}
	}

	var result []map[string]any
	for _, c := range comments {
		result = append(result, commentToHaloComment(c))
	}
	return result
}

func commentToHaloComment(c *models.Comment) map[string]any {
	owner := map[string]any{
		"displayName": c.AuthorName,
		"avatar":      "",
		"annotations": map[string]any{
			"email-hash": md5Hash(strings.ToLower(strings.TrimSpace(c.AuthorEmail))),
		},
	}
	return map[string]any{
		"metadata": map[string]any{
			"name":             strconv.Itoa(c.ID),
			"creationTimestamp": c.CreatedAt.Format(time.RFC3339),
		},
		"spec": map[string]any{
			"content":     c.Content,
			"creationTime": c.CreatedAt.Format(time.RFC3339),
			"owner":       owner,
			"subjectRef": map[string]any{
				"group": "content.halo.run",
				"kind":  "Post",
				"name":  strconv.Itoa(c.ArticleID),
			},
		},
		"owner": owner,
	}
}

// SiteStatsFinder provides site statistics for templates.
type SiteStatsFinder struct {
	articleService  *service.ArticleService
	commentService  *service.CommentService
	categoryService *service.CategoryService
}

func NewSiteStatsFinder(articleSvc *service.ArticleService, commentSvc *service.CommentService, categorySvc *service.CategoryService) *SiteStatsFinder {
	return &SiteStatsFinder{articleService: articleSvc, commentService: commentSvc, categoryService: categorySvc}
}

func (f *SiteStatsFinder) GetStats() map[string]any {
	ctx := context.Background()
	postCount := 0
	if _, total, err := f.articleService.PublicList(ctx, 0, 1, "", ""); err == nil {
		postCount = total
	}
	commentCount := 0
	if f.commentService != nil {
		if total, err := f.commentService.Count(ctx); err == nil {
			commentCount = total
		}
	}
	categoryCount := 0
	if f.categoryService != nil {
		if cats, err := f.categoryService.List(ctx); err == nil {
			categoryCount = len(cats)
		}
	}
	return map[string]any{
		"post":     postCount,
		"comment":  commentCount,
		"category": categoryCount,
		"upvote":   0,
		"visit":    0,
	}
}

// String 方法便于调试
func (p *HaloPost) String() string       { return fmt.Sprintf("Post{%s}", p.Spec.Title) }
func (c *HaloCategory) String() string   { return fmt.Sprintf("Category{%s}", c.Spec.DisplayName) }
func (t *HaloTag) String() string        { return fmt.Sprintf("Tag{%s}", t.Spec.DisplayName) }
func (r *HaloListResult) String() string { return fmt.Sprintf("ListResult{total=%d}", r.Total) }

// toInt converts various numeric types to int.
func toInt(v any) int {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case int:
		return val
	case int8:
		return int(val)
	case int16:
		return int(val)
	case int32:
		return int(val)
	case int64:
		return int(val)
	case uint:
		return int(val)
	case uint8:
		return int(val)
	case uint16:
		return int(val)
	case uint32:
		return int(val)
	case uint64:
		return int(val)
	case float32:
		return int(val)
	case float64:
		return int(val)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(val))
		if err == nil {
			return n
		}
	}
	return 0
}

func md5Hash(s string) string {
	h := md5.Sum([]byte(s))
	return hex.EncodeToString(h[:])
}
