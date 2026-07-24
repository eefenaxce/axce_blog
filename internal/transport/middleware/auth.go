package middleware

import (
	"log"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/eefenaxce/axce_blog/internal/utils"
)

type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

func errorResponse(c fiber.Ctx, status int, msg string) error {
	return c.Status(status).JSON(Response{
		Code: status,
		Data: nil,
		Msg:  msg,
	})
}

type AuthMiddleware struct {
	jwtManager  *utils.JWTManager
	redisClient *utils.RedisClient
}

func NewAuthMiddleware(jwtManager *utils.JWTManager, redisClient *utils.RedisClient) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager:  jwtManager,
		redisClient: redisClient,
	}
}

func (m *AuthMiddleware) RequireAuth() fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return errorResponse(c, fiber.StatusUnauthorized, "missing authorization header")
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return errorResponse(c, fiber.StatusUnauthorized, "invalid authorization header format")
		}

		token := parts[1]

		if m.redisClient != nil {
			blacklisted, _ := m.redisClient.IsBlacklisted(c.RequestCtx(), token)
			if blacklisted {
				return errorResponse(c, fiber.StatusUnauthorized, "token has been revoked")
			}
		}

		claims, err := m.jwtManager.Validate(token)
		if err != nil {
			log.Printf("Invalid token: %v", err)
			return errorResponse(c, fiber.StatusUnauthorized, "invalid token")
		}

		c.Locals("userID", claims.UserID)
		c.Locals("username", claims.Username)
		c.Locals("group", claims.Group)
		c.Locals("token", token)

		return c.Next()
	}
}

func (m *AuthMiddleware) RequireAdmin() fiber.Handler {
	return func(c fiber.Ctx) error {
		group := c.Locals("group")
		if group != "admin" {
			return errorResponse(c, fiber.StatusForbidden, "admin access required")
		}
		return c.Next()
	}
}

func RateLimiter(redisClient *utils.RedisClient) fiber.Handler {
	return func(c fiber.Ctx) error {
		if redisClient == nil {
			return c.Next()
		}

		ip := c.IP()
		key := "rate_limit:" + ip

		count, _ := redisClient.Get(c.RequestCtx(), key)
		if count != "" && count > "60" {
			return errorResponse(c, fiber.StatusTooManyRequests, "rate limit exceeded")
		}

		redisClient.Increment(c.RequestCtx(), key, 60)

		return c.Next()
	}
}
