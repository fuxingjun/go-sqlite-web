package main

import (
	"embed"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/fuxingjun/go-sqlite-web/app/routes"
	"github.com/fuxingjun/go-sqlite-web/app/utils"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func setupRoutes(app *fiber.App) {
	routes.AuthRoute(app)
	routes.DatabaseRoute(app)
	routes.TableRoute(app)
}

//go:embed vue3-sqlite-web/dist/*
var distFS embed.FS

func setupStatic(app *fiber.App) {
	app.Use("/", filesystem.New(filesystem.Config{
		Root:         http.FS(distFS),
		PathPrefix:   "vue3-sqlite-web/dist",
		Browse:       true,
		NotFoundFile: "vue3-sqlite-web/dist/index.html",
		MaxAge:       31536000, // 1年 - 适用于哈希文件名
	}))
}

func main() {

	db := flag.String("db", "db/test_db.sqlite", "SQLite database file")
	host := flag.String("host", "127.0.0.1", "Server host")
	port := flag.Int("port", 12249, "Server port")
	readonly := flag.Bool("readonly", false, "Open database in read-only mode")
	debug := flag.Bool("debug", false, "Enable debug mode with detailed logging")

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
