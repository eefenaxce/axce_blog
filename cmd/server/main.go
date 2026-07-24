package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"

	"github.com/eefenaxce/axce_blog/internal/config"
	"github.com/eefenaxce/axce_blog/internal/db"
	"github.com/eefenaxce/axce_blog/internal/repository"
	"github.com/eefenaxce/axce_blog/internal/service"
	"github.com/eefenaxce/axce_blog/internal/transport"
	"github.com/eefenaxce/axce_blog/internal/transport/handler"
	"github.com/eefenaxce/axce_blog/internal/utils"
)

func main() {
	// Load configuration
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	database, err := db.NewPostgres(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Initialize Redis
	var redisClient *utils.RedisClient
	redis, err := db.NewRedis(cfg.Redis)
	if err != nil {
		log.Printf("Warning: Failed to connect to Redis: %v", err)
	} else {
		redisClient = utils.NewRedisClient(redis)
		defer redis.Close()
		// 每次启动清空 Redis（主题下载进度等临时数据）
		_ = redisClient.FlushDB(context.Background())
	}

	// Initialize utilities
	jwtManager := utils.NewJWTManager(cfg.JWT.Secret, cfg.JWT.ExpirationHours)
	emailSender := utils.NewEmailSender(cfg.Email)

	// Initialize repositories
	dbPool := repository.NewDB(database.Pool)
	userRepo := repository.NewUserRepository(dbPool)
	articleRepo := repository.NewArticleRepository(dbPool)
	articleUpvoteRepo := repository.NewArticleUpvoteRepository(dbPool)
	categoryRepo := repository.NewCategoryRepository(dbPool)
	tagRepo := repository.NewTagRepository(dbPool)
	articleTagRepo := repository.NewArticleTagRepository(dbPool)
	articleCategoryRepo := repository.NewArticleCategoryRepository(dbPool)
	commentRepo := repository.NewCommentRepository(dbPool)
	settingRepo := repository.NewSettingRepository(dbPool)
	menuRepo := repository.NewMenuRepository(dbPool)

	// Initialize services
	settingService := service.NewSettingService(settingRepo)
	verificationService := service.NewVerificationService(redisClient, emailSender)
	userService := service.NewUserService(userRepo, jwtManager, redisClient, emailSender, verificationService, settingService)
	articleService := service.NewArticleService(articleRepo, articleUpvoteRepo, tagRepo, articleTagRepo, articleCategoryRepo, categoryRepo, redisClient)
	categoryService := service.NewCategoryService(categoryRepo, articleRepo, redisClient)
	tagService := service.NewTagService(tagRepo, articleTagRepo, redisClient)
	commentService := service.NewCommentService(commentRepo, articleRepo, settingService)
	menuService := service.NewMenuService(menuRepo)
	themeService := service.NewThemeService(settingService, redisClient)

	// Initialize handlers
	userHandler := handler.NewUserHandler(userService)
	articleHandler := handler.NewArticleHandler(articleService)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	tagHandler := handler.NewTagHandler(tagService)
	commentHandler := handler.NewCommentHandler(commentService)
	haloCommentHandler := handler.NewHaloCommentHandler(commentService, articleService, userService, settingService, jwtManager)
	haloSearchHandler := handler.NewHaloSearchHandler()
	haloUpvoteHandler := handler.NewHaloUpvoteHandler(articleService, jwtManager)
	settingHandler := handler.NewSettingHandler(settingService, redisClient)
	themeHandler := handler.NewThemeHandler(themeService)

	// Initialize theme renderer
	themeRenderer := transport.NewThemeRenderer(
		themeService, settingService,
		articleService, categoryService, tagService, userService,
		menuService, commentService,
	)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "github.com/eefenaxce/axce_blog",
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,
		IdleTimeout:  time.Second * 120,
		BodyLimit:    50 * 1024 * 1024, // 50MB for theme upload
	})

	// Setup API routes first (highest priority)
	transport.SetupRouter(
		app,
		jwtManager,
		redisClient,
		userHandler,
		articleHandler,
		categoryHandler,
		tagHandler,
		commentHandler,
		haloCommentHandler,
		haloSearchHandler,
		haloUpvoteHandler,
		settingHandler,
		themeHandler,
	)

	// Serve static assets (CSS, JS, images) from SPA build
	app.Use("/static", static.New("./web/build/client/static"))

	// Theme static assets (CSS, JS, images) — must be before SPA fallback
	app.Get("/themes/:id/*", themeRenderer.ServeThemeAsset)

	// Server-side rendered theme pages — must be before SPA fallback
	app.Get("/", themeRenderer.ServeHomepage)
	app.Get("/page/:n", themeRenderer.ServeHomepagePaged)
	app.Get("/archives", themeRenderer.ServeArchives)
	app.Get("/archives/page/:n", themeRenderer.ServeArchivesPaged)
	app.Get("/archives/:slug", themeRenderer.ServePost)
	app.Get("/categories", themeRenderer.ServeCategories)
	app.Get("/categories/:slug", themeRenderer.ServeCategory)
	app.Get("/categories/:slug/page/:n", themeRenderer.ServeCategoryPaged)
	app.Get("/tags", themeRenderer.ServeTags)
	app.Get("/tags/:slug", themeRenderer.ServeTag)
	app.Get("/tags/:slug/page/:n", themeRenderer.ServeTagPaged)
	app.Get("/search", themeRenderer.ServeSearch)

	// SPA fallback for admin & auth
	ssrHandler := transport.NewSSRHandler(settingService, themeService)
	if err := ssrHandler.Load("./web/build/client/index.html"); err != nil {
		log.Printf("Warning: Failed to load SPA index.html: %v", err)
	}
	app.Get("/admin/*", ssrHandler.Serve)
	app.Get("/login", ssrHandler.Serve)
	app.Get("/register", ssrHandler.Serve)
	app.Get("/forgot-password", ssrHandler.Serve)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := app.Listen(":" + cfg.Server.Port); err != nil {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	log.Printf("Server started on port %s", cfg.Server.Port)

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
