package middleware

import (
	"crypto/subtle"
	"strings"

	"github.com/gofiber/fiber/v3"
)

// MinioWebhookMiddleware validates the auth_token MinIO sends as Authorization Bearer.
type MinioWebhookMiddleware struct {
	secret string
}

func NewMinioWebhookMiddleware(secret string) *MinioWebhookMiddleware {
	return &MinioWebhookMiddleware{secret: secret}
}

func (m *MinioWebhookMiddleware) Protected() fiber.Handler {
	return func(c fiber.Ctx) error {
		if m.secret == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "webhook not configured",
			})
		}
		token := bearerToken(c.Get("Authorization"))
		if token == "" {
			token = c.Get("Authorization")
		}
		if subtle.ConstantTimeCompare([]byte(token), []byte(m.secret)) != 1 {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "invalid signature",
			})
		}
		return c.Next()
	}
}

func bearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}
