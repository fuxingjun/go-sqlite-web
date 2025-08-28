package routes

import (
	"fmt"
	"strings"

	"github.com/fuxingjun/go-sqlite-web/app/models"
	"github.com/fuxingjun/go-sqlite-web/app/services"
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
)

// QueryRequest 查询请求
type QueryRequest struct {
	SQL  string `json:"sql"`
	Page int    `json:"page,omitempty"`
	Size int    `json:"size,omitempty"`
}

var validate = validator.New()

func DatabaseRoute(router fiber.Router) {
	// 分组前缀
	group := router.Group("/db")

	group.Get("/info", func(c *fiber.Ctx) error {
		resp, err := services.GetDBInfo()
		if err != nil {
			return c.JSON(models.Err("failed to get db info: " + err.Error()))
		}
		return c.JSON(models.OK(resp, "db info retrieved successfully"))
	})

	group.Get("/tables", func(c *fiber.Ctx) error {
		tables, err := services.GetTables()
		if err != nil {
			return c.JSON(models.Err("failed to load tables: " + err.Error()))
		}
		return c.JSON(models.OK(tables, fmt.Sprintf("%d tables found", len(tables))))
	})

	group.Get("/views", func(c *fiber.Ctx) error {
		views, err := services.GetViews()
		if err != nil {
			return c.JSON(models.Err("failed to load views: " + err.Error()))
		}
		return c.JSON(models.OK(views, fmt.Sprintf("%d views found", len(*views))))
	})

	group.Get("/triggers", func(c *fiber.Ctx) error {
		triggers, err := services.GetAllTriggers()
		if err != nil {
			return c.JSON(models.Err("failed to load triggers: " + err.Error()))
		}
		return c.JSON(models.OK(triggers, fmt.Sprintf("%d triggers found", len(triggers))))
	})
	// 创建表
	group.Post("/table", func(c *fiber.Ctx) error {
		var req models.CreateTableRequest
		// 解析 JSON
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(models.Err("invalid JSON: " + err.Error()))
		}
		// 结构体验证
		if err := validate.Struct(&req); err != nil {
			return c.Status(400).JSON(models.Err("validation error: " + err.Error()))
		}
		// 创建表
		if err := services.CreateSQLiteTable(&req); err != nil {
			return c.JSON(models.Err("failed to create table: " + err.Error()))
		}
		return c.JSON(models.OK(nil, fmt.Sprintf("table '%s' created successfully", req.TableName)))
	})
	// 删除表
	group.Delete("/table/:tableName", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		if err := services.DropSQLiteTable(tableName); err != nil {
			return c.JSON(models.Err("failed to drop table: " + err.Error()))
		}
		return c.JSON(models.OK(nil, fmt.Sprintf("drop table '%s' successfully", tableName)))
	})

	group.Post("/query", func(c *fiber.Ctx) error {
		var req QueryRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(models.Err("invalid request"))
		}
		if req.Page < 1 {
			req.Page = 1
		}
		if req.Size < 1 || req.Size > 1000 {
			req.Size = 1000
		}
		result, err := services.ExecuteQuery(req.SQL, req.Page, req.Size) // 默认 1000 条/页
		if err != nil {
			return c.JSON(models.Err("query failed: " + err.Error()))
		}
		return c.JSON(models.OK(result, "query executed"))
	})

	group.Post("/export", func(c *fiber.Ctx) error {
		var req QueryRequest
		if err := c.BodyParser(&req); err != nil {
			return c.Status(400).JSON(models.Err("invalid request"))
		}
		// 校验 SQL 等逻辑...
		if !strings.HasPrefix(strings.ToLower(strings.TrimSpace(req.SQL)), "select") {
			return c.Status(400).JSON(models.Err("only SELECT allowed"))
		}
		fileType := c.Query("type", "json")
		filename, contentType := "data.json", "application/json; charset=utf-8"
		if fileType == "csv" {
			filename, contentType = "data.csv", "text/csv; charset=utf-8"
		}
		c.Set("Content-Type", contentType)
		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		// 获取 *bufio.Writer
		bw := c.Context().Response.BodyWriter()
		err := services.ExportQuery(req.SQL, req.Page, req.Size, fileType, bw)
		// 注意：ExportQuery 内部会使用 csv.NewWriter 或 json.NewEncoder
		// 它们会 flush 到 bw，而 bw 会在 handler 结束时自动 flush（或你 defer）
		return err // 如果导出函数返回 error，Fiber 会处理
	})

}
