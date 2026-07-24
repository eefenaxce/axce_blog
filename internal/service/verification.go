package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/eefenaxce/axce_blog/internal/utils"
)

// CodePurpose identifies the use case for a verification code.
type CodePurpose string

const (
	CodePurposeRegister       CodePurpose = "register"
	CodePurposeForgotPassword CodePurpose = "forgot_password"
)

// CodeConfig is a self-contained description of how to generate, store,
// rate-limit, and deliver a verification code. All hardcoded strings live
// here so callers can inject their own without modifying generic logic.
type CodeConfig struct {
	Purpose      CodePurpose
	Length       int           // for numeric codes; ignored if Value is set
	Value        string        // pre-generated value; if empty, auto-generate numeric code
	TTL          time.Duration // code validity duration
	RateTTL      time.Duration // minimum interval between sends
	RateMsg      string        // error when rate-limited
	EmailSubject string
	EmailBody    string // supports %s placeholder for the code/token
}

// VerificationService is a standalone service for sending and verifying
// one-time codes/tokens. It has no dependency on UserService or any
// specific business domain — just Redis and email.
type VerificationService struct {
	redisClient *utils.RedisClient
	emailSender *utils.EmailSender
}

func NewVerificationService(redisClient *utils.RedisClient, emailSender *utils.EmailSender) *VerificationService {
	return &VerificationService{
		redisClient: redisClient,
		emailSender: emailSender,
	}
}

// codeKey returns the Redis key for storing a verification code/token.
func (v *VerificationService) codeKey(purpose CodePurpose, email string) string {
	return fmt.Sprintf("code:%s:%s", purpose, email)
}

// rateKey returns the Redis key for rate-limiting sends.
func (v *VerificationService) rateKey(purpose CodePurpose, email string) string {
	return fmt.Sprintf("code_rate:%s:%s", purpose, email)
}

// Send performs the full code-delivery flow: rate check → generate/store → email.
func (v *VerificationService) Send(ctx context.Context, email string, cfg CodeConfig) error {
	// ── Rate limit ──
	if v.redisClient != nil {
		exists, _ := v.redisClient.Get(ctx, v.rateKey(cfg.Purpose, email))
		if exists != "" {
			return errors.New(cfg.RateMsg)
		}
	}

	// ── Value (code or token) ──
	value := cfg.Value
	if value == "" {
		var err error
		value, err = generateCode(cfg.Length)
		if err != nil {
			return err
		}
	}

	// ── Store ──
	if v.redisClient != nil {
		v.redisClient.Set(ctx, v.codeKey(cfg.Purpose, email), value, cfg.TTL)
		v.redisClient.Set(ctx, v.rateKey(cfg.Purpose, email), "1", cfg.RateTTL)
	}

	// ── Email ──
	if v.emailSender != nil {
		body := fmt.Sprintf(cfg.EmailBody, value)
		go func() {
			if err := v.emailSender.Send(email, cfg.EmailSubject, body); err != nil {
				log.Printf("Verification email failed (%s → %s): %v", cfg.Purpose, email, err)
			}
		}()
	}

	return nil
}

// Verify checks a user-supplied code against the stored value. On success
// the code is consumed (deleted) so it cannot be reused.
func (v *VerificationService) Verify(ctx context.Context, email string, code string, purpose CodePurpose) error {
	if v.redisClient == nil {
		return errors.New("验证服务暂不可用")
	}

	key := v.codeKey(purpose, email)
	stored, err := v.redisClient.Get(ctx, key)
	if err != nil || stored == "" {
		return errors.New("验证码已过期，请重新获取")
	}
	if stored != code {
		return errors.New("验证码不匹配")
	}

	// One-time use — consume immediately.
	v.redisClient.Delete(ctx, key)
	return nil
}
