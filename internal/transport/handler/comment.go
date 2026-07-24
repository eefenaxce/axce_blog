package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/eefenaxce/axce_blog/internal/service"
)

type CommentHandler struct {
	commentService *service.CommentService
}

func NewCommentHandler(commentService *service.CommentService) *CommentHandler {
	return &CommentHandler{commentService: commentService}
}

type CreateCommentRequest struct {
	ArticleID   int    `json:"article_id"`
	ParentID    *int   `json:"parent_id,omitempty"`
	AuthorName  string `json:"author_name"`
	AuthorEmail string `json:"author_email"`
	AuthorURL   string `json:"author_url"`
	Content     string `json:"content"`
}

// Create handles anonymous/public comment creation (Halo-style).
func (h *CommentHandler) Create(c fiber.Ctx) error {
	var req CreateCommentRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	input := service.CreateCommentInput{
		ArticleID:   req.ArticleID,
		ParentID:    req.ParentID,
		AuthorName:  req.AuthorName,
		AuthorEmail: req.AuthorEmail,
		AuthorURL:   req.AuthorURL,
		Content:     req.Content,
		IPAddress:   c.IP(),
	}

	// If the caller is authenticated (optional JWT), attach the user.
	if uid, ok := c.Locals("userID").(int); ok {
		input.UserID = &uid
		if input.AuthorName == "" {
			if username, ok := c.Locals("username").(string); ok && username != "" {
				input.AuthorName = username
			}
		}
	}

	comment, err := h.commentService.Create(c.Context(), input)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Created(c, comment, "")
}

func (h *CommentHandler) Delete(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid comment id")
	}

	userID, _ := c.Locals("userID").(int)
	group, _ := c.Locals("group").(string)
	isAdmin := group == "admin"

	if err := h.commentService.Delete(c.Context(), id, userID, isAdmin); err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, nil, "comment deleted")
}

func (h *CommentHandler) ListByArticle(c fiber.Ctx) error {
	articleID, err := strconv.Atoi(c.Params("article_id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid article id")
	}

	offset := fiber.Query(c, "offset", 0)
	limit := fiber.Query(c, "limit", 20)

	comments, total, err := h.commentService.ListByArticle(c.Context(), articleID, offset, limit)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return Success(c, fiber.Map{
		"items":   comments,
		"total":   total,
		"offset":  offset,
		"limit":   limit,
	}, "")
}

// AdminList lists all comments for moderation (Halo-style admin comment API).
func (h *CommentHandler) AdminList(c fiber.Ctx) error {
	offset := fiber.Query(c, "offset", 0)
	limit := fiber.Query(c, "limit", 20)
	status := strings.TrimSpace(c.Query("status"))

	comments, total, err := h.commentService.List(c.Context(), offset, limit, status)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return Success(c, fiber.Map{
		"items":  comments,
		"total":  total,
		"offset": offset,
		"limit":  limit,
	}, "")
}

type UpdateCommentStatusRequest struct {
	Status string `json:"status"`
}

// UpdateStatus approves or rejects a comment (admin only).
func (h *CommentHandler) UpdateStatus(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid comment id")
	}

	var req UpdateCommentStatusRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	switch req.Status {
	case "approved":
		err = h.commentService.Approve(c.Context(), id)
	case "rejected":
		err = h.commentService.Reject(c.Context(), id)
	default:
		return Error(c, fiber.StatusBadRequest, "invalid status")
	}

	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, nil, "comment status updated")
}
