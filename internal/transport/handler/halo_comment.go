package handler

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/eefenaxce/axce_blog/internal/models"
	"github.com/eefenaxce/axce_blog/internal/service"
	"github.com/eefenaxce/axce_blog/internal/utils"
)

// HaloCommentHandler exposes Halo 2.x-compatible comment APIs for the
// built-in comment widget (`halo-comment` web component). The widget calls
// endpoints under `/apis/api.plugin.halo.run/v1alpha1/plugins/PluginCommentWidget`.
type HaloCommentHandler struct {
	commentService *service.CommentService
	articleService *service.ArticleService
	userService    *service.UserService
	settingService *service.SettingService
	jwtManager     *utils.JWTManager
}

func NewHaloCommentHandler(
	commentService *service.CommentService,
	articleService *service.ArticleService,
	userService *service.UserService,
	settingService *service.SettingService,
	jwtManager *utils.JWTManager,
) *HaloCommentHandler {
	return &HaloCommentHandler{
		commentService: commentService,
		articleService: articleService,
		userService:    userService,
		settingService: settingService,
		jwtManager:     jwtManager,
	}
}

// List returns approved comments for a subject (Post or SinglePage).
// Example: /apis/api.plugin.halo.run/v1alpha1/plugins/PluginCommentWidget/comments?group=content.halo.run&kind=Post&name=slug
func (h *HaloCommentHandler) List(c fiber.Ctx) error {
	group := c.Query("group", "content.halo.run")
	kind := c.Query("kind", "Post")
	name := strings.TrimSpace(c.Query("name"))

	if name == "" {
		return Success(c, fiber.Map{
			"items":       []any{},
			"total":       0,
			"page":        1,
			"size":        20,
			"totalPages":  0,
			"hasNext":     false,
			"hasPrevious": false,
		}, "")
	}

	if !h.isCommentAllowed(c.Context(), name) {
		return Success(c, fiber.Map{
			"items":       []any{},
			"total":       0,
			"page":        1,
			"size":        20,
			"totalPages":  0,
			"hasNext":     false,
			"hasPrevious": false,
		}, "")
	}

	article, err := h.articleService.GetBySlug(c.Context(), name)
	if err != nil || article == nil {
		return Success(c, fiber.Map{
			"items":       []any{},
			"total":       0,
			"page":        1,
			"size":        20,
			"totalPages":  0,
			"hasNext":     false,
			"hasPrevious": false,
		}, "")
	}

	page, _ := strconv.Atoi(c.Query("page", "1"))
	size, _ := strconv.Atoi(c.Query("size", "20"))
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	offset := (page - 1) * size

	comments, total, err := h.commentService.ListByArticle(c.Context(), article.ID, offset, size)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}

	totalPages := total / size
	if total%size > 0 {
		totalPages++
	}

	items := make([]any, 0, len(comments))
	for _, comment := range comments {
		items = append(items, toHaloCommentVO(comment, group, kind, name))
	}

	return Success(c, fiber.Map{
		"items":       items,
		"total":       total,
		"page":        page,
		"size":        size,
		"totalPages":  totalPages,
		"hasNext":     page < totalPages,
		"hasPrevious": page > 1,
	}, "")
}

// Create creates a comment from the Halo comment widget.
func (h *HaloCommentHandler) Create(c fiber.Ctx) error {
	var req struct {
		Spec struct {
			Content string `json:"content"`
			Owner   struct {
				DisplayName string `json:"displayName"`
				Email       string `json:"email"`
				Website     string `json:"website"`
			} `json:"owner"`
			SubjectRef struct {
				Group string `json:"group"`
				Kind  string `json:"kind"`
				Name  string `json:"name"`
			} `json:"subjectRef"`
		} `json:"spec"`
		ParentID *int `json:"parentId"`
	}
	if err := c.Bind().JSON(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	name := strings.TrimSpace(req.Spec.SubjectRef.Name)
	if name == "" {
		return Error(c, fiber.StatusBadRequest, "subject name is required")
	}

	if !h.isCommentAllowed(c.Context(), name) {
		return Error(c, fiber.StatusForbidden, "comments are disabled")
	}

	article, err := h.articleService.GetBySlug(c.Context(), name)
	if err != nil || article == nil {
		return Error(c, fiber.StatusNotFound, "post not found")
	}

	input := service.CreateCommentInput{
		ArticleID:   article.ID,
		ParentID:    req.ParentID,
		AuthorName:  strings.TrimSpace(req.Spec.Owner.DisplayName),
		AuthorEmail: strings.TrimSpace(req.Spec.Owner.Email),
		AuthorURL:   strings.TrimSpace(req.Spec.Owner.Website),
		Content:     strings.TrimSpace(req.Spec.Content),
		IPAddress:   c.IP(),
	}

	if claims, ok := currentUserFromRequest(c, h.jwtManager); ok {
		input.UserID = &claims.UserID
		if user, err := h.userService.GetByID(c.Context(), claims.UserID); err == nil && user != nil {
			if input.AuthorName == "" {
				if user.Nickname != "" {
					input.AuthorName = user.Nickname
				} else {
					input.AuthorName = user.Username
				}
			}
			input.AuthorAvatar = user.Avatar
		}
	}

	comment, err := h.commentService.Create(c.Context(), input)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, toHaloCommentVO(comment, req.Spec.SubjectRef.Group, req.Spec.SubjectRef.Kind, name), "")
}

// Render returns the Halo comment widget as an HTML page.
// The <halo-comment> custom element embeds this page so the comment UI is
// provided by the backend rather than built inline in the theme.
func (h *HaloCommentHandler) Render(c fiber.Ctx) error {
	group := c.Query("group", "content.halo.run")
	kind := c.Query("kind", "Post")
	name := strings.TrimSpace(c.Query("name"))
	ctx := c.Context()

	allowComment := h.isCommentAllowed(ctx, name)

	var comments []map[string]any
	if allowComment && name != "" {
		if article, err := h.articleService.GetBySlug(ctx, name); err == nil && article != nil {
			list, _, _ := h.commentService.ListByArticle(ctx, article.ID, 0, 1000)
			for _, cm := range list {
				comments = append(comments, toHaloCommentVO(cm, group, kind, name))
			}
		}
	}
	if comments == nil {
		comments = []map[string]any{}
	}

	var currentUser map[string]any
	token := tokenFromRequest(c)
	if claims, ok := currentUserFromRequest(c, h.jwtManager); ok {
		currentUser = map[string]any{
			"userId":   claims.UserID,
			"username": claims.Username,
			"nickname": claims.Nickname,
			"avatar":   claims.Avatar,
			"group":    claims.Group,
		}
	}

	requireReview := "false"
	if rr, _ := h.settingService.Get(ctx, "require_review"); rr == "true" {
		requireReview = "true"
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(renderCommentWidgetHTML(group, kind, name, allowComment, comments, currentUser, token, requireReview))
}

// GetConfig returns the comment widget configuration for a subject.
// The widget uses this to decide whether comments are allowed.
// The :name parameter is treated as the post slug so per-post comment toggles
// are honored; if no matching post is found, the global setting is used.
func (h *HaloCommentHandler) GetConfig(c fiber.Ctx) error {
	name := strings.TrimSpace(c.Params("name"))
	allowComment := h.isCommentAllowed(c.Context(), name)

	return Success(c, fiber.Map{
		"allowComment":  allowComment,
		"requireReview": false,
		"version":       "1.0.0",
	}, "")
}

// Delete removes a comment. Only the comment owner or an administrator may
// delete a comment.
func (h *HaloCommentHandler) Delete(c fiber.Ctx) error {
	name := strings.TrimSpace(c.Params("name"))
	id, err := strconv.Atoi(name)
	if err != nil || id <= 0 {
		return Error(c, fiber.StatusBadRequest, "invalid comment id")
	}

	claims, ok := currentUserFromRequest(c, h.jwtManager)
	if !ok {
		return Error(c, fiber.StatusUnauthorized, "unauthorized")
	}

	if err := h.commentService.Delete(c.Context(), id, claims.UserID, false); err != nil {
		if errors.Is(err, service.ErrCommentNotFound) {
			return Error(c, fiber.StatusNotFound, "comment not found")
		}
		return Error(c, fiber.StatusForbidden, err.Error())
	}

	return Success(c, nil, "")
}

// isCommentAllowed returns true when comments are globally enabled and the
// referenced post (if any) also has its own comment toggle enabled.
func (h *HaloCommentHandler) isCommentAllowed(ctx context.Context, name string) bool {
	enabled, _ := h.settingService.Get(ctx, "enable_comments")
	if enabled == "false" {
		return false
	}

	if name == "" {
		return true
	}

	article, err := h.articleService.GetBySlug(ctx, name)
	if err != nil || article == nil {
		return true
	}
	return article.CommentEnabled
}

func toHaloCommentVO(c *models.Comment, group, kind, subjectName string) map[string]any {
	owner := map[string]any{
		"displayName": c.AuthorName,
		"email":       c.AuthorEmail,
		"website":     c.AuthorURL,
	}
	if c.UserID != nil {
		owner["userId"] = *c.UserID
	}
	if c.AuthorAvatar != "" {
		owner["avatar"] = c.AuthorAvatar
	}
	vo := map[string]any{
		"id": c.ID,
		"metadata": map[string]any{
			"name":              strconv.Itoa(c.ID),
			"creationTimestamp": c.CreatedAt.Format(time.RFC3339),
		},
		"spec": map[string]any{
			"content":      c.Content,
			"creationTime": c.CreatedAt.Format(time.RFC3339),
			"owner":        owner,
			"subjectRef": map[string]any{
				"group": group,
				"kind":  kind,
				"name":  subjectName,
			},
		},
		"owner": owner,
	}
	if c.AuthorAvatar != "" {
		vo["avatar"] = c.AuthorAvatar
	}
	if c.ParentID != nil {
		vo["parentId"] = *c.ParentID
	}
	return vo
}

// renderCommentWidgetHTML returns the full HTML page for the Halo comment widget.
func renderCommentWidgetHTML(group, kind, name string, allowComment bool, comments []map[string]any, currentUser map[string]any, token, requireReview string) string {
	commentsJSON, _ := json.Marshal(comments)
	currentUserJSON, _ := json.Marshal(currentUser)
	allowStr := "false"
	if allowComment {
		allowStr = "true"
	}
	closedDisplay := "none"
	formDisplay := "block"
	if !allowComment {
		closedDisplay = "block"
		formDisplay = "none"
	}

	jsGroup := jsStringLiteral(group)
	jsKind := jsStringLiteral(kind)
	jsName := jsStringLiteral(name)
	jsToken := jsStringLiteral(token)
	jsRequireReview := "false"
	if requireReview == "true" {
		jsRequireReview = "true"
	}
	jsCurrentUser := string(currentUserJSON)
	if jsCurrentUser == "null" {
		jsCurrentUser = "null"
	}

	var b strings.Builder
	b.WriteString(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>评论</title>
  <style>
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: var(--halo-comment-widget-base-font-family, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif);
      color: var(--halo-comment-widget-base-color, #333);
      background: transparent;
      font-size: 14px;
      line-height: 1.6;
      -webkit-font-smoothing: antialiased;
    }
    .halo-comment-widget { padding: 0; }
    .hc-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 20px;
      padding-bottom: 12px;
      border-bottom: 1px solid var(--halo-comment-widget-base-border-color, #eee);
    }
    .hc-title {
      margin: 0;
      font-size: 18px;
      font-weight: 600;
      color: var(--halo-comment-widget-base-color, #333);
    }
    .comment-list { margin-bottom: 24px; }
    .comment-item {
      display: flex;
      gap: 14px;
      padding: 18px 0;
      border-bottom: 1px solid var(--halo-comment-widget-base-border-color, #f0f0f0);
    }
    .comment-item:last-child { border-bottom: none; }
    .comment-avatar {
      width: 42px;
      height: 42px;
      border-radius: 50%;
      background: var(--halo-comment-widget-base-border-color, #eee);
      flex-shrink: 0;
      overflow: hidden;
      box-shadow: 0 0 0 1px rgba(0,0,0,0.04);
    }
    .comment-avatar img { width: 100%; height: 100%; object-fit: cover; }
    .comment-body { flex: 1; min-width: 0; }
    .comment-header {
      display: flex;
      align-items: center;
      gap: 10px;
      margin-bottom: 6px;
      flex-wrap: wrap;
    }
    .comment-author {
      font-weight: 600;
      color: var(--halo-comment-widget-base-color, #333);
      text-decoration: none;
      font-size: 14px;
    }
    .comment-author:hover { color: var(--main, #3b82f6); }
    .comment-time { color: #999; font-size: 12px; }
    .comment-content {
      word-break: break-word;
      color: var(--halo-comment-widget-base-color, #444);
      font-size: 14px;
      line-height: 1.75;
    }
    .comment-content p { margin: 0 0 10px; }
    .comment-content p:last-child { margin-bottom: 0; }
    .comment-actions { margin-top: 10px; }
    .comment-actions button {
      background: none;
      border: none;
      color: #999;
      cursor: pointer;
      padding: 0;
      font-size: 13px;
      margin-right: 14px;
      transition: color 0.2s;
    }
    .comment-actions button:last-child { margin-right: 0; }
    .comment-actions button:hover { color: var(--main, #3b82f6); }
    .comment-actions .hc-delete { color: #999; }
    .comment-actions .hc-delete:hover { color: #e74c3c; }
    .comment-children {
      margin-left: 56px;
      margin-top: 12px;
      padding-left: 16px;
      border-left: 2px solid var(--halo-comment-widget-base-border-color, #f0f0f0);
    }
    .comment-children .comment-item {
      padding: 14px 0;
      border-bottom: none;
    }
    .comment-children .comment-item:first-child { padding-top: 0; }
    .comment-children .comment-item:last-child { padding-bottom: 0; }
    .comment-form {
      display: `)
	b.WriteString(formDisplay)
	b.WriteString(`;
      background: var(--halo-comment-widget-form-bg-color, #f8f9fa);
      border: 1px solid var(--halo-comment-widget-base-border-color, #edf0f2);
      border-radius: 12px;
      padding: 20px;
      margin-bottom: 28px;
    }
    #hc-content:focus {
      outline: none;
      border-color: var(--main, #3b82f6);
      box-shadow: 0 0 0 3px rgba(59,130,246,0.1);
    }
    #hc-content {
      width: 100%;
      min-height: 100px;
      padding: 12px 14px;
      border: 1px solid var(--halo-comment-widget-base-border-color, #e1e4e8);
      border-radius: 8px;
      font-family: inherit;
      font-size: 14px;
      background: #fff;
      color: inherit;
      resize: vertical;
      transition: border-color 0.2s, box-shadow 0.2s;
    }
    .form-actions { display: flex; justify-content: flex-end; align-items: center; gap: 12px; margin-top: 12px; }
    .form-actions button {
      padding: 9px 22px;
      border: none;
      border-radius: 8px;
      background: var(--main, #3b82f6);
      color: #fff;
      cursor: pointer;
      font-size: 14px;
      font-weight: 500;
      transition: opacity 0.2s, transform 0.1s;
    }
    .form-actions button:hover:not(:disabled) { opacity: 0.9; }
    .form-actions button:active:not(:disabled) { transform: scale(0.98); }
    .form-actions button:disabled { opacity: 0.6; cursor: not-allowed; }
    .form-actions button[type="button"] {
      background: transparent;
      color: var(--halo-comment-widget-base-color, #666);
    }
    .form-actions button[type="button"]:hover { background: var(--halo-comment-widget-base-border-color, #f0f0f0); opacity: 1; }
    .logged-in-info {
      display: none;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 14px;
    }
    .hc-user { display: flex; align-items: center; gap: 10px; }
    .hc-user img {
      width: 38px;
      height: 38px;
      border-radius: 50%;
      object-fit: cover;
      background: #eee;
      border: 2px solid #fff;
      box-shadow: 0 1px 3px rgba(0,0,0,0.08);
    }
    .hc-avatar-placeholder {
      width: 42px;
      height: 42px;
      border-radius: 50%;
      background: linear-gradient(135deg, var(--main, #3b82f6), #60a5fa);
      color: #fff;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 15px;
      font-weight: 600;
      border: 2px solid #fff;
      box-shadow: 0 1px 3px rgba(0,0,0,0.08);
    }
    .hc-user-name { font-size: 14px; color: var(--halo-comment-widget-base-color, #333); font-weight: 600; line-height: 1.3; }
    .hc-user-meta { font-size: 12px; color: #999; line-height: 1.3; }
    .hc-logout { font-size: 13px; color: #999; cursor: pointer; text-decoration: none; transition: color 0.2s; }
    .hc-logout:hover { color: var(--main, #3b82f6); }
    .reply-tip {
      font-size: 13px;
      color: var(--main, #3b82f6);
      margin-bottom: 10px;
      display: none;
      background: rgba(59,130,246,0.08);
      padding: 6px 10px;
      border-radius: 6px;
    }
    .empty, .loading, .error, .closed {
      padding: 40px 0;
      text-align: center;
      color: #999;
      font-size: 14px;
    }
    .empty::before {
      content: "💬";
      display: block;
      font-size: 32px;
      margin-bottom: 10px;
      opacity: 0.5;
    }
    .error { color: #e74c3c; }
    .closed {
      display: `)
	b.WriteString(closedDisplay)
	b.WriteString(`;
      color: #999;
      padding: 24px 0;
      text-align: center;
    }
    .pagination {
      display: none;
      justify-content: center;
      gap: 8px;
      margin-top: 16px;
    }
    .pagination button {
      padding: 6px 12px;
      border: 1px solid var(--halo-comment-widget-base-border-color, #ddd);
      background: #fff;
      border-radius: 6px;
      cursor: pointer;
      color: inherit;
      transition: border-color 0.2s, color 0.2s;
    }
    .pagination button:hover:not(:disabled) { border-color: var(--main, #3b82f6); color: var(--main, #3b82f6); }
    .pagination button:disabled { opacity: 0.5; cursor: not-allowed; }
    .hc-modal {
      position: fixed;
      inset: 0;
      z-index: 1000;
      display: flex;
      align-items: center;
      justify-content: center;
    }
    .hc-modal-overlay {
      position: absolute;
      inset: 0;
      background: rgba(0,0,0,0.45);
      backdrop-filter: blur(2px);
    }
    .hc-modal-content {
      position: relative;
      z-index: 1;
      width: min(360px, 92vw);
      background: #fff;
      border-radius: 12px;
      padding: 22px;
      box-shadow: 0 20px 60px rgba(0,0,0,0.2);
      animation: hc-modal-in 0.2s ease-out;
    }
    @keyframes hc-modal-in {
      from { opacity: 0; transform: scale(0.95) translateY(10px); }
      to { opacity: 1; transform: scale(1) translateY(0); }
    }
    .hc-modal-title {
      margin: 0 0 8px;
      font-size: 16px;
      font-weight: 600;
      color: var(--halo-comment-widget-base-color, #333);
    }
    .hc-modal-body {
      margin: 0 0 20px;
      font-size: 14px;
      color: #666;
      line-height: 1.5;
    }
    .hc-modal-actions {
      display: flex;
      justify-content: flex-end;
      gap: 10px;
    }
    .hc-btn {
      padding: 8px 18px;
      border: none;
      border-radius: 8px;
      font-size: 14px;
      cursor: pointer;
      transition: opacity 0.2s, transform 0.1s;
    }
    .hc-btn:hover { opacity: 0.9; }
    .hc-btn:active { transform: scale(0.98); }
    .hc-btn-secondary {
      background: var(--halo-comment-widget-base-border-color, #f0f0f0);
      color: var(--halo-comment-widget-base-color, #666);
    }
    .hc-btn-danger {
      background: #e74c3c;
      color: #fff;
    }
    .hc-toast {
      position: fixed;
      top: 16px;
      left: 50%;
      transform: translateX(-50%) translateY(-20px);
      z-index: 2000;
      padding: 10px 18px;
      border-radius: 8px;
      font-size: 14px;
      color: #fff;
      background: rgba(0,0,0,0.75);
      backdrop-filter: blur(4px);
      opacity: 0;
      pointer-events: none;
      transition: opacity 0.2s, transform 0.2s;
      max-width: min(400px, 90vw);
      text-align: center;
    }
    .hc-toast.show {
      opacity: 1;
      transform: translateX(-50%) translateY(0);
    }
    .hc-toast.error { background: rgba(231,76,60,0.9); }
  </style>
</head>
<body>
  <div class="halo-comment-widget" id="halo-comment">
    <div class="closed">博主已关闭当前页面的评论</div>
    <div class="hc-header">
      <h3 class="hc-title">评论 <span class="hc-count" id="hc-count">0</span></h3>
    </div>
    <div class="comment-form" id="hc-form">
      <div class="logged-in-info" id="hc-logged-in-info"></div>
      <div class="reply-tip" id="hc-reply-tip"></div>
      <textarea id="hc-content" placeholder="写下你的评论..."></textarea>
      <div class="form-actions">
        <button type="button" id="hc-cancel-reply" style="display:none">取消回复</button>
        <button type="submit" id="hc-submit">发表评论</button>
      </div>
    </div>
    <div class="comment-list" id="hc-list"><div class="loading">加载评论中...</div></div>
    <div class="pagination" id="hc-pagination"></div>
  </div>

  <div class="hc-modal" id="hc-delete-modal" style="display:none;">
    <div class="hc-modal-overlay"></div>
    <div class="hc-modal-content">
      <h4 class="hc-modal-title">删除评论</h4>
      <p class="hc-modal-body">确定要删除这条评论吗？删除后无法恢复。</p>
      <div class="hc-modal-actions">
        <button type="button" class="hc-btn hc-btn-secondary" id="hc-cancel-delete">取消</button>
        <button type="button" class="hc-btn hc-btn-danger" id="hc-confirm-delete">删除</button>
      </div>
    </div>
  </div>

  <div class="hc-toast" id="hc-toast"></div>

  <script>
    window.__hc_token = `)
	b.WriteString(jsToken)
	b.WriteString(`;
    window.__hc_requireReview = `)
	b.WriteString(jsRequireReview)
	b.WriteString(`;
    window.__hc_currentUser = `)
	b.WriteString(jsCurrentUser)
	b.WriteString(`;
    (function () {
      const cfg = {
        group: `)
	b.WriteString(jsGroup)
	b.WriteString(`,
        kind: `)
	b.WriteString(jsKind)
	b.WriteString(`,
        name: `)
	b.WriteString(jsName)
	b.WriteString(`,
        allowComment: `)
	b.WriteString(allowStr)
	b.WriteString(`,
        comments: `)
	b.Write(commentsJSON)
	b.WriteString(`
      };

      const token = window.__hc_token || '';
      const requireReview = window.__hc_requireReview === true;
      let currentUser = window.__hc_currentUser || null;
      const info = document.getElementById('hc-logged-in-info');
      if (currentUser) {
        if (info) {
          info.style.display = 'flex';
          const name = escapeHtml(currentUser.nickname || currentUser.username || '登录用户');
          const avatar = currentUser.avatar ? escapeHtml(currentUser.avatar) : '';
          const avatarHtml = avatar ? '<img src="' + avatar + '" alt="" />' : '<span class="hc-avatar-placeholder">' + name.charAt(0) + '</span>';
          info.innerHTML = '<div class="hc-user">' + avatarHtml + '<div><div class="hc-user-name">' + name + '</div><div class="hc-user-meta">已登录</div></div></div>' +
            '<a id="hc-logout" class="hc-logout">退出登录</a>';
          document.getElementById('hc-logout').addEventListener('click', () => {
            localStorage.removeItem('auth_token');
            location.reload();
          });
        }
      } else {
        if (info) {
          info.style.display = 'flex';
          info.innerHTML = '<div class="hc-user"><span class="hc-avatar-placeholder">?</span><div><div class="hc-user-name">未登录</div><div class="hc-user-meta">请先登录后发表评论</div></div></div>';
        }
        const submitBtn = document.getElementById('hc-submit');
        if (submitBtn) {
          submitBtn.disabled = true;
          submitBtn.textContent = '登录后评论';
        }
      }
      function authHeaders() {
        const h = { 'Content-Type': 'application/json' };
        if (token) h['Authorization'] = 'Bearer ' + token;
        return h;
      }
      function canDelete(node) {
        if (!currentUser) return false;
        const owner = node.owner || (node.spec && node.spec.owner) || {};
        return owner.userId != null && Number(owner.userId) === Number(currentUser.userId);
      }

      function escapeHtml(str) {
        if (str == null) return '';
        return String(str)
          .replace(/&/g, '&amp;')
          .replace(/</g, '&lt;')
          .replace(/>/g, '&gt;')
          .replace(/"/g, '&quot;')
          .replace(/'/g, '&#39;');
      }
      function showToast(message, type) {
        const toast = document.getElementById('hc-toast');
        if (!toast) return;
        toast.textContent = message;
        toast.className = 'hc-toast' + (type === 'error' ? ' error' : '');
        toast.classList.add('show');
        if (toast._timer) clearTimeout(toast._timer);
        toast._timer = setTimeout(() => toast.classList.remove('show'), 3000);
      }
      function formatTime(iso) {
        if (!iso) return '';
        const d = new Date(iso);
        if (isNaN(d.getTime())) return iso;
        const pad = (n) => String(n).padStart(2, '0');
        return d.getFullYear() + '-' + pad(d.getMonth()+1) + '-' + pad(d.getDate()) + ' ' + pad(d.getHours()) + ':' + pad(d.getMinutes());
      }
      function ownerName(node) {
        const owner = node.owner || (node.spec && node.spec.owner) || {};
        return owner.displayName || owner.name || node.displayName || node.authorName || '匿名';
      }
      function ownerUrl(node) {
        const owner = node.owner || (node.spec && node.spec.owner) || {};
        return owner.website || owner.url || node.authorUrl || node.website || '';
      }
      function buildTree(items) {
        const map = {};
        const roots = [];
        items.forEach((item) => { item.children = []; map[String(item.id)] = item; });
        items.forEach((item) => {
          if (item.parentId != null && map[String(item.parentId)]) {
            map[String(item.parentId)].children.push(item);
          } else {
            roots.push(item);
          }
        });
        return roots;
      }
      function renderNode(node) {
        const item = document.createElement('div');
        item.className = 'comment-item';
        item.id = 'comment-' + node.id;
        const owner = node.owner || (node.spec && node.spec.owner) || {};
        const authorName = owner.displayName || owner.name || node.displayName || node.authorName || '匿名';
        const authorUrl = owner.website || owner.url || node.authorUrl || node.website || '';
        const avatarUrl = owner.avatar || node.avatar || '';
        const authorLink = authorUrl
          ? '<a class="comment-author" href="' + escapeHtml(authorUrl) + '" target="_blank" rel="nofollow">' + escapeHtml(authorName) + '</a>'
          : '<span class="comment-author">' + escapeHtml(authorName) + '</span>';
        const avatar = avatarUrl ? '<img src="' + escapeHtml(avatarUrl) + '" alt="" />' : '<span class="hc-avatar-placeholder" style="width:40px;height:40px;font-size:14px;">' + escapeHtml(authorName.charAt(0)) + '</span>';
        const deleteBtn = canDelete(node) ? '<button class="hc-delete" data-id="' + node.id + '">删除</button>' : '';
        item.innerHTML = '<div class="comment-avatar">' + avatar + '</div>'
          + '<div class="comment-body">'
          + '<div class="comment-header">' + authorLink + '<span class="comment-time">' + formatTime(node.createTime || node.creationTime || node.createdAt) + '</span></div>'
          + '<div class="comment-content">' + (node.content || (node.spec && node.spec.content) || '') + '</div>'
          + '<div class="comment-actions"><button class="hc-reply" data-id="' + node.id + '" data-name="' + escapeHtml(authorName) + '">回复</button>' + deleteBtn + '</div>'
          + '<div class="comment-children"></div>'
          + '</div>';
        item.querySelector('.hc-reply').addEventListener('click', (e) => {
          setReply(Number(e.target.getAttribute('data-id')), e.target.getAttribute('data-name'));
        });
        const delEl = item.querySelector('.hc-delete');
        if (delEl) {
          delEl.addEventListener('click', (e) => {
            showDeleteModal(Number(e.target.getAttribute('data-id')));
          });
        }
        const childrenEl = item.querySelector('.comment-children');
        (node.children || []).forEach((child) => childrenEl.appendChild(renderNode(child)));
        return item;
      }
      let replyTo = null;
      let pendingDeleteId = null;
      function showDeleteModal(id) {
        pendingDeleteId = id;
        document.getElementById('hc-delete-modal').style.display = 'flex';
      }
      function hideDeleteModal() {
        pendingDeleteId = null;
        document.getElementById('hc-delete-modal').style.display = 'none';
      }
      function setReply(parentId, name) {
        replyTo = parentId;
        const tip = document.getElementById('hc-reply-tip');
        tip.textContent = '回复 @' + name;
        tip.style.display = 'block';
        document.getElementById('hc-cancel-reply').style.display = 'inline-block';
        document.getElementById('hc-content').focus();
      }
      function cancelReply() {
        replyTo = null;
        document.getElementById('hc-reply-tip').style.display = 'none';
        document.getElementById('hc-cancel-reply').style.display = 'none';
      }
      async function deleteComment(id) {
        try {
          const res = await fetch('/apis/api.plugin.halo.run/v1alpha1/plugins/PluginCommentWidget/comments/' + id, {
            method: 'DELETE',
            headers: authHeaders()
          });
          if (!res.ok) {
            const err = await res.json().catch(() => ({}));
            throw new Error(err.message || '删除失败 (' + res.status + ')');
          }
          cfg.comments = cfg.comments.filter((c) => c.id !== id && Number(c.id) !== id);
          renderList();
        } catch (e) {
          showToast('删除失败：' + e.message, 'error');
        }
      }
      function renderList() {
        const listEl = document.getElementById('hc-list');
        const countEl = document.getElementById('hc-count');
        listEl.innerHTML = '';
        if (countEl) countEl.textContent = cfg.comments.length;
        if (!cfg.comments.length) {
          listEl.innerHTML = '<div class="empty">暂无评论，快来抢沙发吧~</div>';
          return;
        }
        buildTree(cfg.comments).forEach((node) => listEl.appendChild(renderNode(node)));
      }
      async function submit(e) {
        e.preventDefault();
        if (!currentUser) return showToast('请先登录后发表评论', 'error');
        const content = document.getElementById('hc-content').value.trim();
        if (!content) return showToast('请填写评论内容', 'error');
        const btn = document.getElementById('hc-submit');
        btn.disabled = true;
        btn.textContent = '提交中...';
        try {
          const body = {
            spec: {
              allowNotification: false,
              content,
              subjectRef: { group: cfg.group, kind: cfg.kind, name: cfg.name }
            },
            parentId: replyTo || undefined
          };
          const res = await fetch('/apis/api.plugin.halo.run/v1alpha1/plugins/PluginCommentWidget/comments', {
            method: 'POST',
            headers: authHeaders(),
            body: JSON.stringify(body)
          });
          if (!res.ok) {
            const err = await res.json().catch(() => ({}));
            throw new Error(err.message || '提交失败 (' + res.status + ')');
          }
          document.getElementById('hc-content').value = '';
          cancelReply();
          const created = await res.json();
          const vo = created.data || created;
          if (vo && (vo.status === 'pending' || requireReview)) {
            showToast('评论已提交，等待管理员审核', 'success');
          } else {
            cfg.comments.unshift(vo);
            renderList();
            showToast('评论提交成功', 'success');
          }
        } catch (e) {
          showToast('评论提交失败：' + e.message, 'error');
        } finally {
          btn.disabled = false;
          btn.textContent = '发表评论';
        }
      }
      document.getElementById('hc-cancel-delete').addEventListener('click', hideDeleteModal);
      document.getElementById('hc-confirm-delete').addEventListener('click', () => {
        if (pendingDeleteId != null) {
          deleteComment(pendingDeleteId);
          hideDeleteModal();
        }
      });
      document.querySelector('#hc-delete-modal .hc-modal-overlay').addEventListener('click', hideDeleteModal);
      if (cfg.allowComment) {
        document.getElementById('hc-submit').addEventListener('click', submit);
        document.getElementById('hc-cancel-reply').addEventListener('click', cancelReply);
        renderList();
      }
    })();
    (function () {
      const params = new URLSearchParams(location.search);
      const frameId = params.get('frameId');
      if (!frameId || !parent.postMessage) return;
      function reportHeight() {
        parent.postMessage({ type: 'halo-comment-resize', frameId: frameId, height: document.body.scrollHeight }, '*');
      }
      window.addEventListener('load', reportHeight);
      if (typeof ResizeObserver !== 'undefined') {
        new ResizeObserver(reportHeight).observe(document.body);
      } else {
        window.addEventListener('resize', reportHeight);
      }
      setTimeout(reportHeight, 100);
      setTimeout(reportHeight, 500);
    })();
  </script>
</body>
</html>`)
	return b.String()
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
