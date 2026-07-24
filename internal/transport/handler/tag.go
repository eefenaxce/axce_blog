package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/eefenaxce/axce_blog/internal/service"
)

type TagHandler struct {
	tagService *service.TagService
}

func NewTagHandler(tagService *service.TagService) *TagHandler {
	return &TagHandler{tagService: tagService}
}

type CreateTagRequest struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

func (h *TagHandler) Create(c fiber.Ctx) error {
	var req CreateTagRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	input := service.CreateTagInput{Name: req.Name, Icon: req.Icon}

	tag, err := h.tagService.Create(c.Context(), input)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Created(c, tag, "")
}

func (h *TagHandler) Get(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid tag id")
	}

	tag, err := h.tagService.GetByID(c.Context(), id)
	if err != nil {
		return Error(c, fiber.StatusNotFound, err.Error())
	}

	return Success(c, tag, "")
}

type UpdateTagRequest struct {
	Name string `json:"name"`
	Icon string `json:"icon"`
}

func (h *TagHandler) Update(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid tag id")
	}

	var req UpdateTagRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	input := service.UpdateTagInput{Name: req.Name, Icon: req.Icon}

	tag, err := h.tagService.Update(c.Context(), id, input)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, tag, "tag updated")
}

func (h *TagHandler) Delete(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid tag id")
	}

	if err := h.tagService.Delete(c.Context(), id); err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, nil, "tag deleted")
}

func (h *TagHandler) List(c fiber.Ctx) error {
	tags, err := h.tagService.List(c.Context())
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return Success(c, fiber.Map{"tags": tags}, "")
}

// PublicList 公开的标签列表
func (h *TagHandler) PublicList(c fiber.Ctx) error {
	tags, err := h.tagService.List(c.Context())
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}
	return Success(c, fiber.Map{"tags": tags}, "")
}

// fiber:context-methods migrated
