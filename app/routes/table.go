package routes

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fuxingjun/go-sqlite-web/app/models"
	"github.com/fuxingjun/go-sqlite-web/app/services"
	"github.com/gofiber/fiber/v2"
)

func TableRoute(router fiber.Router) {
	// 分组前缀
	group := router.Group("/table")

	// 查询表信息
	group.Get("/:tableName", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		if tableName == "" {
			return c.Status(400).JSON(models.Err("tableName is required"))
		}
		info, err := services.GetTableInfo(tableName)
		if err != nil {
			return c.JSON(models.Err("failed to get table info: " + err.Error()))
		}
		if len(info.Columns) == 0 {
			return c.Status(404).JSON(models.Err("table not found or has no columns"))
		}
		return c.JSON(models.OK(info, "table info retrieved successfully"))
	})

	// 查询表字段
	group.Get("/:tableName/columns", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		indexes, err := services.GetTableColumns(tableName)
		if err != nil {
			return c.JSON(models.Err("failed to get table indexes: " + err.Error()))
		}
		return c.JSON(models.OK(indexes, "table columns retrieved successfully"))
	})

	// 新建表字段
	group.Post("/:tableName/columns", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		var column services.NewTableColumnSchema
		if err := c.BodyParser(&column); err != nil {
			return c.Status(400).JSON(models.Err("invalid JSON body: " + err.Error()))
		}
		if column.Name == "" || column.Type == "" {
			return c.Status(400).JSON(models.Err("column name and type are required"))
		}
		err := services.NewTableColumn(tableName, column)
		if err != nil {
			return c.JSON(models.Err("failed to add column: " + err.Error()))
		}
		return c.JSON(models.OK(nil, "column added successfully"))
	})

	// 删除表字段
	group.Delete("/:tableName/columns/:columnName", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		columnName := c.Params("columnName")
		if err := services.DeleteTableColumn(tableName, columnName); err != nil {
			return c.JSON(models.Err("failed to delete column: " + err.Error()))
		}
		return c.JSON(models.OK(nil, "column deleted successfully"))
	})

	// 表字段重命名
	group.Put("/:tableName/columns/:columnName", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		columnName := c.Params("columnName")
		var body struct {
			NewName string `json:"newName"`
		}
		if err := c.BodyParser(&body); err != nil {
			return c.Status(400).JSON(models.Err("invalid JSON body: " + err.Error()))
		}
		if body.NewName == "" {
			return c.Status(400).JSON(models.Err("new column name is required"))
		}
		if err := services.RenameTableColumn(tableName, columnName, body.NewName); err != nil {
			return c.JSON(models.Err("failed to rename column: " + err.Error()))
		}
		return c.JSON(models.OK(nil, "column renamed successfully"))
	})

	// 查询表索引
	group.Get("/:tableName/indexes", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		indexes, err := services.GetTableIndexes(tableName)
		if err != nil {
			return c.JSON(models.Err("failed to get table indexes: " + err.Error()))
		}
		return c.JSON(models.OK(indexes, "table indexes retrieved successfully"))
	})

	// 新建表索引
	group.Post("/:tableName/indexes", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		var index services.NewTableIndexSchema
		if err := c.BodyParser(&index); err != nil {
			return c.Status(400).JSON(models.Err("invalid JSON body: " + err.Error()))
		}
		if index.Name == "" || len(index.Columns) == 0 {
			return c.Status(400).JSON(models.Err("index name and columns are required"))
		}
		err := services.NewTableIndex(tableName, index)
		if err != nil {
			return c.JSON(models.Err("failed to add index: " + err.Error()))
		}
		return c.JSON(models.OK(nil, "index added successfully"))
	})

	// 删除表索引
	group.Delete("/:tableName/indexes/:indexName", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		indexName := c.Params("indexName")
		if err := services.DeleteTableIndex(tableName, indexName); err != nil {
			return c.JSON(models.Err("failed to delete index: " + err.Error()))
		}
		return c.JSON(models.OK(nil, "index deleted successfully"))
	})

	// 查询表数据
	group.Get("/:tableName/rows", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		page, _ := strconv.Atoi(c.Query("page", "1"))
		limit, _ := strconv.Atoi(c.Query("limit", "50"))
		if page <= 0 {
			page = 1
		}
		if limit <= 0 {
			limit = 50
		}
		offset := (page - 1) * limit

		resp, err := services.GetTableData(tableName, limit, offset)
		if err != nil {
			return c.JSON(models.Err("failed to get table data: " + err.Error()))
		}

		return c.JSON(models.OK(map[string]any{
			"rows":       resp.Data,
			"total":      resp.Total,
			"page":       page,
			"totalPages": (resp.Total + limit - 1) / limit,
			"limit":      limit,
		}, ""))
	})

	// 新建数据行
	group.Post("/:tableName/row", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		var data map[string]any
		if err := c.BodyParser(&data); err != nil {
			return c.Status(400).JSON(models.Err("invalid JSON body: " + err.Error()))
		}
		id, err := services.InsertRow(tableName, data)
		if err != nil {
			return c.JSON(models.Err("insert failed: " + err.Error()))
		}
		return c.Status(201).JSON(models.OK(map[string]any{
			"id": id,
		}, "row inserted successfully"))
	})

	// 如果有主键, 支持修改数据
	group.Put("/:tableName/row", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		var data map[string]any
		if err := c.BodyParser(&data); err != nil {
			return c.Status(400).JSON(models.Err("invalid JSON body: " + err.Error()))
		}
		res, err := services.UpdateRow(tableName, data)
		if err != nil {
			return c.JSON(models.Err("update failed: " + err.Error()))
		}
		return c.JSON(models.OK(map[string]any{
			"rowsAffected": res,
		}, "row updated successfully"))
	})

	// 如果有主键, 支持修改数据
	group.Delete("/:tableName/row", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		// 获取所有查询参数作为 map[string]string
		data := c.Queries()
		res, err := services.DeleteRow(tableName, data)
		if err != nil {
			return c.JSON(models.Err("delete failed: " + err.Error()))
		}
		return c.JSON(models.OK(map[string]any{
			"rowsAffected": res,
		}, "row deleted successfully"))
	})

	// 上传导入数据
	group.Post("/:tableName/import", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		if tableName == "" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "缺少参数: table",
			})
		}
		file, err := c.FormFile("file")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "请上传文件",
			})
		}
		ext := strings.ToLower(filepath.Ext(file.Filename))
		if ext != ".json" && ext != ".csv" {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "仅支持 .json 或 .csv 文件",
			})
		}
		fileReader, err := file.Open()
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
				"error": "无法读取文件",
			})
		}
		defer fileReader.Close()
		createNewColumn := c.FormValue("createNewColumn", "true") != "false"
		rollback := c.FormValue("rollback", "false") == "true"
		result, err := services.ImportToTable(
			c.Context(),
			fileReader,
			ext,
			tableName,
			createNewColumn,
			rollback,
		)
		if err != nil {
			return c.JSON(models.ErrWithData(err.Error(), result))
		}
		return c.JSON(models.OK(result, "数据导入完成"))
	})

	// 导出表格数据
	group.Post("/:tableName/export", func(c *fiber.Ctx) error {
		tableName := c.Params("tableName")
		if tableName == "" {
			return c.Status(400).JSON(models.Err("tableName is required"))
		}
		// 校验表名和列名
		if !services.IsValidIdentifier(tableName) {
			return c.Status(400).JSON(models.Err("非法表名"))
		}

		schema := new(models.ExportTableSchema)
		if err := c.BodyParser(schema); err != nil {
			return c.Status(400).JSON(models.Err("invalid JSON body: " + err.Error()))
		}

		fileType := strings.ToLower(schema.FileType)
		if fileType != "json" && fileType != "csv" {
			fileType = "json"
		}

		columns := schema.Columns
		if len(columns) == 0 {
			return c.Status(400).JSON(models.Err("至少指定一个列"))
		}

		for _, col := range columns {
			if !services.IsValidIdentifier(col) {
				return c.Status(400).JSON(models.Err("非法列名: " + col))
			}
		}

		// 设置响应头
		filename := fmt.Sprintf("%s.%s", tableName, fileType)
		contentType := "application/json; charset=utf-8"
		if fileType == "csv" {
			contentType = "text/csv; charset=utf-8"
		}
		c.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
		c.Set("Content-Type", contentType)

		bw := c.Context().Response.BodyWriter()
		err := services.StreamExportTableData(
			c.Context(),
			tableName,
			columns,
			fileType,
			bw,
		)

		return err // 如果导出函数返回 error，Fiber 会处理
	})
}
