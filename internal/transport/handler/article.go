package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/eefenaxce/axce_blog/internal/service"
)

type ArticleHandler struct {
	articleService *service.ArticleService
}

func NewArticleHandler(articleService *service.ArticleService) *ArticleHandler {
	return &ArticleHandler{articleService: articleService}
}

type CreateArticleRequest struct {
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	Content        string `json:"content"`
	CoverURL       string `json:"coverUrl"`
	CategoryIDs    []int  `json:"categoryIds"`
	TagIDs         []int  `json:"tagIds"`
	Status         string `json:"status"`
	CommentEnabled *bool  `json:"commentEnabled"`
}

func (h *ArticleHandler) Create(c fiber.Ctx) error {
	userID := c.Locals("userID").(int)

	var req CreateArticleRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	input := service.CreateArticleInput{
		Title:          req.Title,
		Summary:        req.Summary,
		Content:        req.Content,
		CoverURL:       req.CoverURL,
		CategoryIDs:    req.CategoryIDs,
		TagIDs:         req.TagIDs,
		Status:         req.Status,
		CommentEnabled: req.CommentEnabled,
		UserID:         userID,
	}

	article, err := h.articleService.Create(c.Context(), input)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Created(c, article, "")
}

func (h *ArticleHandler) Get(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid article id")
	}

	article, err := h.articleService.GetByID(c.Context(), id)
	if err != nil {
		return Error(c, fiber.StatusNotFound, err.Error())
	}

	categories, _ := h.articleService.GetCategories(c.Context(), id)
	tags, _ := h.articleService.GetTags(c.Context(), id)

	return Success(c, fiber.Map{
		"article":    article,
		"categories": categories,
		"tags":       tags,
	}, "")
}

func (h *ArticleHandler) GetBySlug(c fiber.Ctx) error {
	slug := c.Params("slug")

	article, err := h.articleService.GetBySlug(c.Context(), slug)
	if err != nil {
		return Error(c, fiber.StatusNotFound, err.Error())
	}

	categories, _ := h.articleService.GetCategories(c.Context(), article.ID)
	tags, _ := h.articleService.GetTags(c.Context(), article.ID)

	return Success(c, fiber.Map{
		"article":    article,
		"categories": categories,
		"tags":       tags,
	}, "")
}

type UpdateArticleRequest struct {
	Title          string `json:"title"`
	Summary        string `json:"summary"`
	Content        string `json:"content"`
	CoverURL       string `json:"coverUrl"`
	CategoryIDs    []int  `json:"categoryIds"`
	TagIDs         []int  `json:"tagIds"`
	Status         string `json:"status"`
	CommentEnabled *bool  `json:"commentEnabled"`
}

func (h *ArticleHandler) Update(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid article id")
	}

	var req UpdateArticleRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	input := service.UpdateArticleInput{
		Title:          req.Title,
		Summary:        req.Summary,
		Content:        req.Content,
		CoverURL:       req.CoverURL,
		CategoryIDs:    req.CategoryIDs,
		TagIDs:         req.TagIDs,
		Status:         req.Status,
		CommentEnabled: req.CommentEnabled,
	}

	article, err := h.articleService.Update(c.Context(), id, input)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, article, "article updated")
}

func (h *ArticleHandler) Delete(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid article id")
	}

	if err := h.articleService.Delete(c.Context(), id); err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, nil, "article deleted")
}

func (h *ArticleHandler) List(c fiber.Ctx) error {
	offset := fiber.Query(c, "offset", 0)
	limit := fiber.Query(c, "limit", 20)

	articles, total, err := h.articleService.List(c.Context(), offset, limit)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return Success(c, fiber.Map{
		"articles": articles,
		"total":    total,
	}, "")
}

func (h *ArticleHandler) ListByUser(c fiber.Ctx) error {
	userID := c.Locals("userID").(int)
	offset := fiber.Query(c, "offset", 0)
	limit := fiber.Query(c, "limit", 20)

	articles, total, err := h.articleService.ListByUser(c.Context(), userID, offset, limit)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return Success(c, fiber.Map{
		"articles": articles,
		"total":    total,
	}, "")
}

// ─── Public Handlers ───

// PublicList 公开文章列表（支持分页、分类slug/标签slug筛选）
func (h *ArticleHandler) PublicList(c fiber.Ctx) error {
	page := 1
	if p := c.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	size := 10
	if s := c.Query("size"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 50 {
			size = n
		}
	}
	offset := (page - 1) * size

	categorySlug := c.Query("category")
	tagSlug := c.Query("tag")

	items, total, err := h.articleService.PublicList(c.Context(), offset, size, categorySlug, tagSlug)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}
	return Success(c, fiber.Map{"items": items, "total": total, "page": page, "size": size}, "")
}

// PublicGet 获取单篇文章（通过 slug）
func (h *ArticleHandler) PublicGet(c fiber.Ctx) error {
	slug := c.Params("slug")
	article, err := h.articleService.GetBySlug(c.Context(), slug)
	if err != nil {
		return Error(c, fiber.StatusNotFound, "文章不存在")
	}
	return Success(c, article, "")
}

// PublicPage 获取自定义页面（通过 slug）
func (h *ArticleHandler) PublicPage(c fiber.Ctx) error {
	slug := c.Params("slug")
	// 把自定义页面当作文章处理（用 same table），或扩展 service
	article, err := h.articleService.GetBySlug(c.Context(), slug)
	if err != nil {
		return Error(c, fiber.StatusNotFound, "页面不存在")
	}
	return Success(c, article, "")
}

// Search searches published articles by keyword (Halo-style public search).
func (h *ArticleHandler) Search(c fiber.Ctx) error {
	keyword := c.Query("keyword")
	if keyword == "" {
		return Success(c, fiber.Map{"items": []any{}, "total": 0, "page": 1, "size": 10}, "")
	}

	page := 1
	if p := c.Query("page"); p != "" {
		if n, err := strconv.Atoi(p); err == nil && n > 0 {
			page = n
		}
	}
	size := 10
	if s := c.Query("size"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 50 {
			size = n
		}
	}
	offset := (page - 1) * size

	items, total, err := h.articleService.Search(c.Context(), keyword, offset, size)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}
	return Success(c, fiber.Map{"items": items, "total": total, "page": page, "size": size}, "")
}
