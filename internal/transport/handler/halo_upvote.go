package handler

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/eefenaxce/axce_blog/internal/service"
	"github.com/eefenaxce/axce_blog/internal/utils"
)

// HaloUpvoteHandler exposes Halo 2.x-compatible tracker endpoints used by themes
// for post upvotes.
type HaloUpvoteHandler struct {
	articleService *service.ArticleService
	jwtManager     *utils.JWTManager
}

func NewHaloUpvoteHandler(articleService *service.ArticleService, jwtManager *utils.JWTManager) *HaloUpvoteHandler {
	return &HaloUpvoteHandler{articleService: articleService, jwtManager: jwtManager}
}

// Upvote toggles an upvote for a subject. The Joe3 theme posts to
// /apis/api.halo.run/v1alpha1/trackers/upvote with {group, plural, name}.
func (h *HaloUpvoteHandler) Upvote(c fiber.Ctx) error {
	var req struct {
		Group  string `json:"group"`
		Plural string `json:"plural"`
		Name   string `json:"name"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return Error(c, fiber.StatusBadRequest, "name is required")
	}

	// Only support post upvotes for now; moments/pages can be added later.
	plural := strings.ToLower(strings.TrimSpace(req.Plural))
	if plural != "posts" && plural != "post" {
		return Error(c, fiber.StatusBadRequest, "unsupported subject type")
	}

	article, err := h.articleService.GetBySlug(c.Context(), name)
	if err != nil || article == nil {
		return Error(c, fiber.StatusNotFound, "post not found")
	}

	userID := 0
	if claims, ok := currentUserFromRequest(c, h.jwtManager); ok {
		userID = claims.UserID
	}

	count, err := h.articleService.ToggleUpvote(c.Context(), article.ID, userID, c.IP())
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return Success(c, fiber.Map{"upvote": count}, "")
}
