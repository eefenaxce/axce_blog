package handler

import (
	"os"
	"path/filepath"

	"github.com/gofiber/fiber/v3"

	"github.com/eefenaxce/axce_blog/internal/service"
	"github.com/eefenaxce/axce_blog/internal/utils"
)

type SettingHandler struct {
	settingService *service.SettingService
	redisClient    *utils.RedisClient
}

func NewSettingHandler(settingService *service.SettingService, redisClient *utils.RedisClient) *SettingHandler {
	return &SettingHandler{settingService: settingService, redisClient: redisClient}
}

type SetSettingRequest struct {
	Key         string `json:"key"`
	Value       string `json:"value"`
	Description string `json:"description"`
}

func (h *SettingHandler) Get(c fiber.Ctx) error {
	key := c.Params("key")

	value, err := h.settingService.Get(c.Context(), key)
	if err != nil {
		return Error(c, fiber.StatusNotFound, "setting not found")
	}

	return Success(c, fiber.Map{"key": key, "value": value}, "")
}

func (h *SettingHandler) Set(c fiber.Ctx) error {
	var req SetSettingRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.settingService.Set(c.Context(), req.Key, req.Value, req.Description); err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	// 清除 SSR 配置缓存，下次请求时会重新加载
	if h.redisClient != nil {
		h.redisClient.Delete(c.Context(), "ssr_config")
	}

	// 清除内存缓存

	return Success(c, nil, "setting updated")
}

func (h *SettingHandler) UploadImage(c fiber.Ctx) error {
	file, err := c.FormFile("image")
	isIcon := false
	if err != nil {
		file, err = c.FormFile("icon")
		if err != nil {
			return Error(c, fiber.StatusBadRequest, "请选择要上传的图片")
		}
		isIcon = true
	}

	if file.Size > 2*1024*1024 {
		return Error(c, fiber.StatusBadRequest, "文件大小不能超过 2MB")
	}

	allowedTypes := []string{"image/png", "image/jpeg", "image/jpg", "image/svg+xml", "image/webp"}
	if !isIcon {
		allowedTypes = append(allowedTypes, "image/gif")
	}
	contentType := file.Header.Get("Content-Type")
	isValid := false
	for _, t := range allowedTypes {
		if t == contentType {
			isValid = true
			break
		}
	}
	if !isValid {
		if isIcon {
			return Error(c, fiber.StatusBadRequest, "只支持 PNG、JPG、SVG、WebP 格式的图片")
		}
		return Error(c, fiber.StatusBadRequest, "只支持图片文件")
	}

	ext := filepath.Ext(file.Filename)
	var filename, savePath, urlPrefix string
	if isIcon {
		filename = "icon_" + utils.GenerateRandomString(16) + ext
		savePath = "./web/build/client/static/icons/"
		urlPrefix = "/static/icons/"
	} else {
		filename = "img_" + utils.GenerateRandomString(16) + ext
		savePath = "./web/build/client/static/images/"
		urlPrefix = "/static/images/"
	}

	if err := os.MkdirAll(savePath, os.ModePerm); err != nil {
		return Error(c, fiber.StatusInternalServerError, "无法创建目录")
	}

	if err := c.SaveFile(file, savePath+filename); err != nil {
		return Error(c, fiber.StatusInternalServerError, "文件保存失败")
	}

	url := urlPrefix + filename

	if isIcon {
		oldIcon, _ := h.settingService.Get(c.Context(), "site_icon")
		if oldIcon != "" {
			oldPath := "./web/build/client/static/icons/" + filepath.Base(oldIcon)
			os.Remove(oldPath)
		}
		if err := h.settingService.Set(c.Context(), "site_icon", url, "网站图标"); err != nil {
			os.Remove(savePath + filename)
			return Error(c, fiber.StatusInternalServerError, "保存设置失败")
		}
		if h.redisClient != nil {
			h.redisClient.Delete(c.Context(), "ssr_config")
		}
		return Success(c, fiber.Map{"icon_url": url}, "图标上传成功")
	}

	return Success(c, fiber.Map{"url": url}, "图片上传成功")
}

func (h *SettingHandler) List(c fiber.Ctx) error {
	settings, err := h.settingService.List(c.Context())
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return Success(c, fiber.Map{"settings": settings}, "")
}

// fiber:context-methods migrated
