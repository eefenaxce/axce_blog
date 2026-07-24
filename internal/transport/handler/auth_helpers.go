package handler

import (
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/eefenaxce/axce_blog/internal/utils"
)

// tokenFromRequest extracts a bearer token from the Authorization header or the
// `token` query/body parameter. It is used by public widget endpoints that need
// optional authentication.
func tokenFromRequest(c fiber.Ctx) string {
	authHeader := c.Get("Authorization")
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) == 2 && parts[0] == "Bearer" {
			return parts[1]
		}
	}
	if t := c.Query("token"); t != "" {
		return t
	}
	if t := c.FormValue("token"); t != "" {
		return t
	}
	return ""
}

// parseToken validates a JWT string using the provided manager.
func parseToken(jwtManager *utils.JWTManager, token string) (*utils.JWTClaims, bool) {
	if jwtManager == nil || token == "" {
		return nil, false
	}
	claims, err := jwtManager.Validate(token)
	if err != nil {
		return nil, false
	}
	return claims, true
}

// currentUserFromRequest returns validated JWT claims from the current request.
func currentUserFromRequest(c fiber.Ctx, jwtManager *utils.JWTManager) (*utils.JWTClaims, bool) {
	return parseToken(jwtManager, tokenFromRequest(c))
}
