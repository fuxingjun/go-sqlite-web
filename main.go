package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/fuxingjun/go-sqlite-web/app/routes"
	"github.com/fuxingjun/go-sqlite-web/app/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func setupRoutes(app *fiber.App) {
	routes.AuthRoute(app)
	routes.DatabaseRoute(app)
	routes.TableRoute(app)

	// 如果前端路由是history模式, 需要 SPA 路由回退
	app.Get("*", func(c *fiber.Ctx) error {
		// 检查是否是 API 请求 /auth /db /table
		if strings.HasPrefix(c.Path(), "/auth") || strings.HasPrefix(c.Path(), "/db") || strings.HasPrefix(c.Path(), "/table") {
			return fiber.ErrNotFound
		}

		// 返回 SPA 的入口文件
		return c.SendFile("./www/index.html")
	})
}

func setupStatic(app *fiber.App) {
	app.Static("/", "./www", fiber.Static{
		// 启用 gzip 压缩
		Compress: true,
		// 支持字节范围请求
		ByteRange: true,
		// 启用目录浏览
		Browse: false,
		Index:  "index.html",
		// MaxAge:        31536000, // 1年 - 适用于哈希文件名
		CacheDuration: 10 * time.Second,

		// 设置安全头
		// ModifyResponse: func(c *fiber.Ctx) error {
		// 	c.Set("X-Content-Type-Options", "nosniff")
		// 	c.Set("X-Frame-Options", "DENY")
		// 	return nil
		// },
	})
}

func main() {

	db := flag.String("db", "", "SQLite database file")
	host := flag.String("host", "127.0.0.1", "Server host")
	port := flag.Int("port", 18808, "Server port")
	readonly := flag.Bool("readonly", false, "Open database in read-only mode")
	debug := flag.Bool("debug", false, "Log ")

	flag.Parse()

	if _, err := os.Stat(*db); os.IsNotExist(err) {
		log.Fatal("Database file does not exist: ", db)
	}
	if err := utils.Connect(*db, *readonly); err != nil {
		log.Fatal("DB connect error: ", err)
	}
	defer utils.DB.Close()
	level := slog.LevelInfo
	if *debug {
		level = slog.LevelDebug
	}
	utils.InitLogger(level, "", "logs", "midnight", 1)

	// 创建 Fiber 应用实例
	app := fiber.New()
	if *debug {
		app.Use(logger.New())
	}
	app.Use(cors.New())
	setupRoutes(app)
	setupStatic(app)

	addr := fmt.Sprintf("%s:%d", *host, *port)
	// 启动服务器在 指定 端口
	fmt.Printf("Listening on %s\n", addr)
	if err := app.Listen(addr); err != nil {
		fmt.Printf("listen failed: %v\n", err)
	}
}
