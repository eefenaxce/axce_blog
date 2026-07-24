package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/eefenaxce/axce_blog/internal/service"
)

type CategoryHandler struct {
	categoryService *service.CategoryService
}

func NewCategoryHandler(categoryService *service.CategoryService) *CategoryHandler {
	return &CategoryHandler{categoryService: categoryService}
}

type CreateCategoryRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

func (h *CategoryHandler) Create(c fiber.Ctx) error {
	var req CreateCategoryRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	input := service.CreateCategoryInput{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Icon:        req.Icon,
	}

	category, err := h.categoryService.Create(c.Context(), input)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Created(c, category, "")
}

func (h *CategoryHandler) Get(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid category id")
	}

	category, err := h.categoryService.GetByID(c.Context(), id)
	if err != nil {
		return Error(c, fiber.StatusNotFound, err.Error())
	}

	return Success(c, category, "")
}

type UpdateCategoryRequest struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

func (h *CategoryHandler) Update(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid category id")
	}

	var req UpdateCategoryRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	input := service.UpdateCategoryInput{
		Name:        req.Name,
		Slug:        req.Slug,
		Description: req.Description,
		Icon:        req.Icon,
	}

	category, err := h.categoryService.Update(c.Context(), id, input)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, category, "category updated")
}

func (h *CategoryHandler) Delete(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid category id")
	}

	deletedCount, err := h.categoryService.Delete(c.Context(), id)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, fiber.Map{
		"articles_count": deletedCount,
	}, "category deleted")
}

func (h *CategoryHandler) List(c fiber.Ctx) error {
	categories, err := h.categoryService.List(c.Context())
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return Success(c, fiber.Map{"categories": categories}, "")
}

// PublicList 公开的分类列表
func (h *CategoryHandler) PublicList(c fiber.Ctx) error {
	categories, err := h.categoryService.List(c.Context())
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}
	return Success(c, fiber.Map{"categories": categories}, "")
}

// fiber:context-methods migrated
