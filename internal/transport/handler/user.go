package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"

	"github.com/eefenaxce/axce_blog/internal/service"
)

type UserHandler struct {
	userService *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{userService: userService}
}

type RegisterRequest struct {
	Username        string `json:"username"`
	Nickname        string `json:"nickname"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
	Code            string `json:"code"`
}

type SendRegisterCodeRequest struct {
	Email string `json:"email"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type ResetPasswordRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (h *UserHandler) SendRegisterCode(c fiber.Ctx) error {
	var req SendRegisterCodeRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.userService.SendRegisterCode(c.Context(), req.Email); err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, nil, "验证码已发送到您的邮箱")
}

func (h *UserHandler) Register(c fiber.Ctx) error {
	var req RegisterRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.Password != req.PasswordConfirm {
		return Error(c, fiber.StatusBadRequest, "两次输入的密码不一致")
	}

	input := service.RegisterInput{
		Username: req.Username,
		Nickname: req.Nickname,
		Email:    req.Email,
		Password: req.Password,
		Code:     req.Code,
	}

	user, token, err := h.userService.Register(c.Context(), input)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Created(c, fiber.Map{
		"user":  user,
		"token": token,
	}, "注册成功")
}

func (h *UserHandler) Login(c fiber.Ctx) error {
	var req LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	input := service.LoginInput{
		Username: req.Username,
		Password: req.Password,
	}

	user, token, err := h.userService.Login(c.Context(), input)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, fiber.Map{
		"user":  user,
		"token": token,
	}, "登录成功")
}

func (h *UserHandler) Logout(c fiber.Ctx) error {
	token := c.Locals("token")
	if token == nil {
		return Error(c, fiber.StatusBadRequest, "no token")
	}

	h.userService.Logout(c.Context(), token.(string))
	return Success(c, nil, "logged out")
}

func (h *UserHandler) GetProfile(c fiber.Ctx) error {
	userID := c.Locals("userID").(int)

	user, err := h.userService.GetByID(c.Context(), userID)
	if err != nil {
		return Error(c, fiber.StatusNotFound, err.Error())
	}

	return Success(c, user, "")
}

type UpdateProfileRequest struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Bio      string `json:"bio"`
}

func (h *UserHandler) UpdateProfile(c fiber.Ctx) error {
	userID := c.Locals("userID").(int)

	var req UpdateProfileRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	input := service.UpdateProfileInput{
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
		Bio:      req.Bio,
	}

	user, err := h.userService.UpdateProfile(c.Context(), userID, input)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, user, "资料已更新")
}

func (h *UserHandler) ListUsers(c fiber.Ctx) error {
	offset := fiber.Query(c, "offset", 0)
	limit := fiber.Query(c, "limit", 20)

	users, total, err := h.userService.List(c.Context(), offset, limit)
	if err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return Success(c, fiber.Map{
		"users": users,
		"total": total,
	}, "")
}

func (h *UserHandler) GetUser(c fiber.Ctx) error {
	username := c.Params("username")

	user, err := h.userService.GetByUsername(c.Context(), username)
	if err != nil {
		return Error(c, fiber.StatusNotFound, err.Error())
	}

	return Success(c, user, "")
}

func (h *UserHandler) UpdateUserStatus(c fiber.Ctx) error {
	userIDStr := c.Params("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid user id")
	}
	statusStr := c.Query("status")

	var status int
	if statusStr == "1" {
		status = 1
	} else {
		status = 0
	}

	if err := h.userService.UpdateStatus(c.Context(), userID, status); err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, nil, "status updated")
}

func (h *UserHandler) DeleteUser(c fiber.Ctx) error {
	userIDStr := c.Params("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid user id")
	}

	if err := h.userService.Delete(c.Context(), userID); err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, nil, "user deleted")
}

func (h *UserHandler) ForgotPassword(c fiber.Ctx) error {
	var req ForgotPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	resetBaseURL := c.Protocol() + "://" + c.Hostname()
	if err := h.userService.ForgotPassword(c.Context(), req.Email, resetBaseURL); err != nil {
		return Error(c, fiber.StatusInternalServerError, err.Error())
	}

	return Success(c, nil, "如果该邮箱已注册，将收到重置密码邮件")
}

func (h *UserHandler) ResetPassword(c fiber.Ctx) error {
	var req ResetPasswordRequest
	if err := c.Bind().Body(&req); err != nil {
		return Error(c, fiber.StatusBadRequest, "invalid request body")
	}

	if err := h.userService.ResetPassword(c.Context(), req.Token, req.Password); err != nil {
		return Error(c, fiber.StatusBadRequest, err.Error())
	}

	return Success(c, nil, "密码重置成功")
}
