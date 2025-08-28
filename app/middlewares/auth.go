package middlewares

import (
	"os"

	"github.com/gofiber/fiber/v2"
)

func AuthRequired() fiber.Handler {
	secret := os.Getenv("API_KEY")
	if secret == "" {
		return func(c *fiber.Ctx) error { return c.Next() } // 未设置则不禁用
	}
	return func(c *fiber.Ctx) error {
		key := c.Get("X-API-Key")
		if key != secret {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"success": false,
				"error":   "Unauthorized",
			})
		}
		return c.Next()
	}
}
