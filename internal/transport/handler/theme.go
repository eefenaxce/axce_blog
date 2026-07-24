package handler

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/eefenaxce/axce_blog/internal/models"
	"github.com/eefenaxce/axce_blog/internal/service"
)

type ThemeHandler struct {
	themeService *service.ThemeService
}

func NewThemeHandler(themeService *service.ThemeService) *ThemeHandler {
	return &ThemeHandler{themeService: themeService}
}

// GetActive 返回当前激活的主题信息（公开接口）
func (h *ThemeHandler) GetActive(c fiber.Ctx) error {
	theme, err := h.themeService.GetActiveTheme(c.Context())
	if err != nil {
		return Error(c, fiber.StatusNotFound, "无激活主题")
	}
	return Success(c, fiber.Map{"theme": theme}, "")
}

func (h *ThemeHandler) Upload(c fiber.Ctx) error {
	file, err := c.FormFile("theme")
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "请选择要上传的主题文件")
	}

	if file.Size > 50*1024*1024 {
		return Error(c, fiber.StatusBadRequest, "文件大小不能超过 50MB")
	}

	fileContent, err := file.Open()
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "无法读取文件")
	}
	defer fileContent.Close()

	theme, err := h.themeService.Upload(fileContent, file.Filename)
	if err != nil {
		if errors.Is(err, service.ErrThemeExists) {
			return Error(c, fiber.StatusConflict, err.Error())
		}
		switch err {
		case service.ErrInvalidTheme:
			return Error(c, fiber.StatusBadRequest, "无效的主题包，未找到 theme.json")
		default:
			return Error(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	return Success(c, theme, "主题上传成功")
}

func (h *ThemeHandler) List(c fiber.Ctx) error {
	themes, err := h.themeService.List(c.Context())
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}
	if themes == nil {
		themes = make([]models.Theme, 0)
	}
	return Success(c, fiber.Map{"themes": themes}, "")
}

func (h *ThemeHandler) Activate(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.themeService.Activate(c.Context(), id); err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}
	return Success(c, nil, "主题已激活")
}

// GetSettings 获取主题设置（表单定义 + 当前值）
func (h *ThemeHandler) GetSettings(c fiber.Ctx) error {
	id := c.Params("id")
	settings, err := h.themeService.GetThemeSettings(c.Context(), id)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}
	return Success(c, settings, "")
}

// UpdateSettings 更新主题设置
func (h *ThemeHandler) UpdateSettings(c fiber.Ctx) error {
	id := c.Params("id")
	var req struct {
		Values map[string]any `json:"values"`
	}
	if err := c.Bind().JSON(&req); err != nil || len(req.Values) == 0 {
		return Error(c, fiber.StatusBadRequest, "请提供要更新的设置值")
	}
	if err := h.themeService.UpdateThemeSetting(c.Context(), id, req.Values); err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}
	return Success(c, nil, "设置已保存")
}

func (h *ThemeHandler) Delete(c fiber.Ctx) error {
	id := c.Params("id")
	if err := h.themeService.Delete(c.Context(), id); err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}
	return Success(c, nil, "主题已删除")
}

func (h *ThemeHandler) FetchRemote(c fiber.Ctx) error {
	keyword := c.Query("keyword", "")
	page, _ := strconv.Atoi(c.Query("page", "0"))
	size, _ := strconv.Atoi(c.Query("size", "20"))

	themes, total, err := h.themeService.FetchRemote(keyword, page, size)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, "获取远程主题失败: "+err.Error())
	}
	if themes == nil {
		themes = make([]models.ThemeListItem, 0)
	}
	return Success(c, fiber.Map{"themes": themes, "total": total}, "")
}

func (h *ThemeHandler) Download(c fiber.Ctx) error {
	var req struct {
		RepoURL string `json:"repoUrl"`
	}
	if err := c.Bind().JSON(&req); err != nil || req.RepoURL == "" {
		return Error(c, fiber.StatusBadRequest, "请提供仓库地址")
	}

	repo, err := h.themeService.DownloadFromURL(req.RepoURL)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, fiber.Map{"repo": repo}, "开始下载")
}

func (h *ThemeHandler) DownloadProgress(c fiber.Ctx) error {
	repos := c.Query("repos", "")
	if repos == "" {
		return Error(c, fiber.StatusBadRequest, "请提供 repo 名称")
	}

	var results []*service.DownloadProgress
	for _, repo := range strings.Split(repos, ",") {
		r := strings.TrimSpace(repo)
		if r == "" {
			continue
		}
		if p := h.themeService.GetDownloadProgress(r); p != nil {
			results = append(results, p)
		}
	}

	return Success(c, fiber.Map{"progresses": results}, "")
}

func (h *ThemeHandler) ServeScreenshot(c fiber.Ctx) error {
	id := c.Params("id")
	path, contentType, err := h.themeService.ServeThemeAsset(id, "screenshot.png")
	if err != nil {
		return Error(c, fiber.StatusNotFound, "screenshot not found")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Error(c, fiber.StatusNotFound, "screenshot not found")
	}
	c.Set("Content-Type", contentType)
	return c.Send(data)
}
