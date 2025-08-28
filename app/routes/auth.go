package routes

import (
	"github.com/fuxingjun/go-sqlite-web/app/models"
	"github.com/gofiber/fiber/v2"
)

func AuthRoute(router fiber.Router) {
	// 分组前缀
	group := router.Group("/auth")

	group.Post("/login", func(c *fiber.Ctx) error {
		return c.JSON(models.OK("data", "login successful"))
	})
	// group.Get("/logout", services.GetTableData)
	// group.Get("/status", services.GetTableData)
}
