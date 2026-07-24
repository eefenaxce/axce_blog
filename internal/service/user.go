package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/eefenaxce/axce_blog/internal/models"
	"github.com/eefenaxce/axce_blog/internal/repository"
	"github.com/eefenaxce/axce_blog/internal/utils"
)

var (
	ErrUserNotFound             = errors.New("用户不存在")
	ErrUserExists               = errors.New("用户已存在")
	ErrInvalidCredentials       = errors.New("用户名或密码错误")
	ErrUserDisabled             = errors.New("用户账号已禁用")
	ErrAccountLocked            = errors.New("账号暂时锁定，请稍后再试")
	ErrPasswordResetUnavailable = errors.New("密码重置功能不可用")
	ErrInvalidResetToken        = errors.New("重置令牌无效或已过期")
)

const (
	defaultUserGroup  = "user"
	defaultUserStatus = 0
)

type UserService struct {
	userRepo       repository.UserRepository
	jwtManager     *utils.JWTManager
	redisClient    *utils.RedisClient
	emailSender    *utils.EmailSender
	verification   *VerificationService
	settingService *SettingService
}

func NewUserService(
	userRepo repository.UserRepository,
	jwtManager *utils.JWTManager,
	redisClient *utils.RedisClient,
	emailSender *utils.EmailSender,
	verification *VerificationService,
	settingService *SettingService,
) *UserService {
	return &UserService{
		userRepo:       userRepo,
		jwtManager:     jwtManager,
		redisClient:    redisClient,
		emailSender:    emailSender,
		verification:   verification,
		settingService: settingService,
	}
}

type RegisterInput struct {
	Username string `validate:"required,min=3,max=50"`
	Nickname string
	Email    string `validate:"required,email"`
	Password string `validate:"required,min=6"`
	Code     string `validate:"required"`
}

func (s *UserService) SendRegisterCode(ctx context.Context, email string) error {
	siteName := s.siteTitle(ctx)
	ttl := 10 * time.Minute
	rateTTL := 60 * time.Second

	return s.verification.Send(ctx, email, CodeConfig{
		Purpose:      CodePurposeRegister,
		Length:       6,
		TTL:          ttl,
		RateTTL:      rateTTL,
		RateMsg:      fmt.Sprintf("验证码已发送，请%d秒后再试", int(rateTTL.Seconds())),
		EmailSubject: fmt.Sprintf("注册验证 - %s", siteName),
		EmailBody:    fmt.Sprintf("您的注册验证码是：%%s\n\n该验证码有效期为%d分钟。\n\n如果不是您本人操作，请忽略此邮件", int(ttl.Minutes())),
	})
}

// siteTitle reads site_title from DB settings; returns empty string on failure.
func (s *UserService) siteTitle(ctx context.Context) string {
	if s.settingService == nil {
		return ""
	}
	v, err := s.settingService.Get(ctx, "site_title")
	if err != nil {
		return ""
	}
	return v
}

func generateCode(length int) (string, error) {
	code := ""
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		code += fmt.Sprintf("%d", n.Int64())
	}
	return code, nil
}

func (s *UserService) Register(ctx context.Context, input RegisterInput) (*models.User, string, error) {
	if err := s.verification.Verify(ctx, input.Email, input.Code, CodePurposeRegister); err != nil {
		return nil, "", err
	}

	// Check if user already exists (double-check)
	existingUser, _ := s.userRepo.GetByUsername(ctx, input.Username)
	if existingUser != nil {
		return nil, "", ErrUserExists
	}

	existingEmail, _ := s.userRepo.GetByEmail(ctx, input.Email)
	if existingEmail != nil {
		return nil, "", ErrUserExists
	}

	hashedPassword, err := utils.HashPassword(input.Password)
	if err != nil {
		return nil, "", err
	}

	user := &models.User{
		Username:     input.Username,
		Nickname:     input.Nickname,
		Email:        input.Email,
		Avatar:       utils.GenerateAvatarURL(input.Email),
		PasswordHash: hashedPassword,
		Group:        defaultUserGroup,
		Status:       defaultUserStatus,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", err
	}

	token, err := s.jwtManager.Generate(user.ID, user.Username, user.Nickname, user.Avatar, user.Group)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

type LoginInput struct {
	Username string `validate:"required"`
	Password string `validate:"required"`
}

func (s *UserService) Login(ctx context.Context, input LoginInput) (*models.User, string, error) {
	if s.redisClient != nil {
		blocked, _ := s.redisClient.IsBlocked(ctx, input.Username)
		if blocked {
			return nil, "", ErrAccountLocked
		}
	}

	user, err := s.userRepo.GetByUsername(ctx, input.Username)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}

	if !utils.CheckPassword(input.Password, user.PasswordHash) {
		if s.redisClient != nil {
			s.redisClient.IncrementFailedAttempts(ctx, input.Username)
		}
		return nil, "", ErrInvalidCredentials
	}

	if user.Status == 1 {
		return nil, "", ErrUserDisabled
	}

	token, err := s.jwtManager.Generate(user.ID, user.Username, user.Nickname, user.Avatar, user.Group)
	if err != nil {
		return nil, "", err
	}

	if s.redisClient != nil {
		s.redisClient.ClearFailedAttempts(ctx, input.Username)
	}

	return user, token, nil
}

func (s *UserService) GetByID(ctx context.Context, id int) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (s *UserService) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

type UpdateProfileInput struct {
	Nickname string `validate:"omitempty,max=100"`
	Avatar   string `validate:"omitempty,max=500"`
	Bio      string
}

func (s *UserService) UpdateProfile(ctx context.Context, userID int, input UpdateProfileInput) (*models.User, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, ErrUserNotFound
	}

	if input.Nickname != "" {
		user.Nickname = input.Nickname
	}
	if input.Avatar != "" {
		user.Avatar = input.Avatar
	}
	if input.Bio != "" {
		user.Bio = input.Bio
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	return user, nil
}

func (s *UserService) List(ctx context.Context, offset, limit int) ([]*models.User, int, error) {
	return s.userRepo.List(ctx, offset, limit)
}

func (s *UserService) UpdateStatus(ctx context.Context, userID int, status int) error {
	return s.userRepo.UpdateStatus(ctx, userID, status)
}

func (s *UserService) Delete(ctx context.Context, userID int) error {
	return s.userRepo.Delete(ctx, userID)
}

func (s *UserService) Logout(ctx context.Context, token string) error {
	if s.redisClient != nil {
		return s.redisClient.BlacklistToken(ctx, token, time.Hour*24*7)
	}
	return nil
}

func (s *UserService) ForgotPassword(ctx context.Context, email, resetBaseURL string) error {
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return nil
	}

	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return err
	}
	resetToken := hex.EncodeToString(buf)

	if s.redisClient != nil {
		s.redisClient.Set(ctx, "reset_token:"+resetToken, user.Email, time.Hour)
	}

	siteName := s.siteTitle(ctx)
	resetURL := fmt.Sprintf("%sforgot-password?token=%s", resetBaseURL, resetToken)
	ttl := 10 * time.Minute
	rateTTL := 60 * time.Second

	return s.verification.Send(ctx, email, CodeConfig{
		Purpose:      CodePurposeForgotPassword,
		Value:        resetToken,
		TTL:          ttl,
		RateTTL:      rateTTL,
		RateMsg:      fmt.Sprintf("密码重置邮件已发送，请%d秒后再试", int(rateTTL.Seconds())),
		EmailSubject: fmt.Sprintf("密码重置 - %s", siteName),
		EmailBody:    fmt.Sprintf("请使用以下链接重置密码：\n\n%s\n\n该链接有效期为%d分钟。\n\n如果不是您本人操作，请忽略此邮件", resetURL, int(ttl.Minutes())),
	})
}

func (s *UserService) ResetPassword(ctx context.Context, token, newPassword string) error {
	if s.redisClient == nil {
		return ErrPasswordResetUnavailable
	}

	email, err := s.redisClient.Get(ctx, "reset_token:"+token)
	if err != nil || email == "" {
		return ErrInvalidResetToken
	}

	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		return ErrUserNotFound
	}

	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return err
	}
	user.PasswordHash = hashedPassword

	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	s.redisClient.Delete(ctx, "reset_token:"+token)

	return nil
}
