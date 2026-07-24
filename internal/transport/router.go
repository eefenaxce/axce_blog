package transport

import (
	"github.com/gofiber/fiber/v3"

	"github.com/eefenaxce/axce_blog/internal/transport/handler"
	"github.com/eefenaxce/axce_blog/internal/transport/middleware"
	"github.com/eefenaxce/axce_blog/internal/utils"
)

type Router struct {
	app                *fiber.App
	userHandler        *handler.UserHandler
	articleHandler     *handler.ArticleHandler
	categoryHandler    *handler.CategoryHandler
	tagHandler         *handler.TagHandler
	commentHandler     *handler.CommentHandler
	haloCommentHandler *handler.HaloCommentHandler
	haloSearchHandler  *handler.HaloSearchHandler
	haloUpvoteHandler  *handler.HaloUpvoteHandler
	settingHandler     *handler.SettingHandler
	themeHandler       *handler.ThemeHandler
	authMiddleware     *middleware.AuthMiddleware
}

func NewRouter(
	userHandler *handler.UserHandler,
	articleHandler *handler.ArticleHandler,
	categoryHandler *handler.CategoryHandler,
	tagHandler *handler.TagHandler,
	commentHandler *handler.CommentHandler,
	haloCommentHandler *handler.HaloCommentHandler,
	haloSearchHandler *handler.HaloSearchHandler,
	haloUpvoteHandler *handler.HaloUpvoteHandler,
	settingHandler *handler.SettingHandler,
	themeHandler *handler.ThemeHandler,
	authMiddleware *middleware.AuthMiddleware,
) *Router {
	return &Router{
		userHandler:        userHandler,
		articleHandler:     articleHandler,
		categoryHandler:    categoryHandler,
		tagHandler:         tagHandler,
		commentHandler:     commentHandler,
		haloCommentHandler: haloCommentHandler,
		haloSearchHandler:  haloSearchHandler,
		haloUpvoteHandler:  haloUpvoteHandler,
		settingHandler:     settingHandler,
		themeHandler:       themeHandler,
		authMiddleware:     authMiddleware,
	}
}

func (r *Router) Setup(app *fiber.App) {
	r.app = app

	r.app.Use(middleware.Logger())
	r.app.Use(middleware.Recovery())
	r.app.Use(middleware.CORS())

	// Public theme screenshot — <img> tag can't send auth headers
	r.app.Get("/api/v1/admin/themes/:id/screenshot", r.themeHandler.ServeScreenshot)

	// Public active theme endpoint — for frontend to know current theme
	r.app.Get("/api/v1/theme", r.themeHandler.GetActive)

	api := r.app.Group("/api/v1")

	// Public routes
	api.Post("/send-register-code", r.userHandler.SendRegisterCode)
	api.Post("/register", r.userHandler.Register)
	api.Post("/login", r.userHandler.Login)
	api.Post("/forgot-password", r.userHandler.ForgotPassword)
	api.Post("/reset-password", r.userHandler.ResetPassword)

	// Public content routes (articles, categories, tags, pages, search)
	api.Get("/articles", r.articleHandler.PublicList)
	api.Get("/articles/:slug", r.articleHandler.PublicGet)
	api.Get("/categories", r.categoryHandler.PublicList)
	api.Get("/tags", r.tagHandler.PublicList)
	api.Get("/page/:slug", r.articleHandler.PublicPage)
	api.Get("/search", r.articleHandler.Search)
	api.Get("/articles/:id", r.articleHandler.Get)
	api.Get("/articles/slug/:slug", r.articleHandler.GetBySlug)
	api.Get("/categories/:id", r.categoryHandler.Get)
	api.Get("/tags/:id", r.tagHandler.Get)
	api.Get("/articles/:article_id/comments", r.commentHandler.ListByArticle)
	api.Post("/comments", r.commentHandler.Create)
	api.Get("/settings", r.settingHandler.List)
	api.Get("/settings/:key", r.settingHandler.Get)
	api.Get("/u/:username", r.userHandler.GetUser)

	// Halo 2.x built-in comment widget APIs (used by the halo-comment web component).
	apis := r.app.Group("/apis/api.plugin.halo.run/v1alpha1/plugins/PluginCommentWidget")
	apis.Get("/comments", r.haloCommentHandler.List)
	apis.Post("/comments", r.haloCommentHandler.Create)
	apis.Get("/configs/:name", r.haloCommentHandler.GetConfig)
	apis.Get("/comments/render", r.haloCommentHandler.Render)
	apis.Delete("/comments/:name", r.haloCommentHandler.Delete)

	// Halo 2.x built-in search widget page (loaded by window.SearchWidget.open()).
	searchApis := r.app.Group("/apis/api.plugin.halo.run/v1alpha1/plugins/PluginSearchWidget")
	searchApis.Get("/render", r.haloSearchHandler.Render)

	// Halo 2.x tracker endpoint (upvote) used by themes.
	r.app.Post("/apis/api.halo.run/v1alpha1/trackers/upvote", r.haloUpvoteHandler.Upvote)

	// Auth required routes
	auth := api.Group("", r.authMiddleware.RequireAuth())
	auth.Post("/logout", r.userHandler.Logout)
	auth.Get("/profile", r.userHandler.GetProfile)
	auth.Put("/profile", r.userHandler.UpdateProfile)
	auth.Get("/admin/articles", r.articleHandler.List)
	auth.Post("/articles", r.articleHandler.Create)
	auth.Put("/articles/:id", r.articleHandler.Update)
	auth.Delete("/articles/:id", r.articleHandler.Delete)
	auth.Delete("/comments/:id", r.commentHandler.Delete)

	// Admin routes
	admin := auth.Group("", r.authMiddleware.RequireAdmin())
	admin.Get("/admin/users", r.userHandler.ListUsers)
	admin.Put("/admin/users/:id/status", r.userHandler.UpdateUserStatus)
	admin.Delete("/admin/users/:id", r.userHandler.DeleteUser)
	admin.Post("/admin/categories", r.categoryHandler.Create)
	admin.Put("/admin/categories/:id", r.categoryHandler.Update)
	admin.Delete("/admin/categories/:id", r.categoryHandler.Delete)
	admin.Post("/admin/tags", r.tagHandler.Create)
	admin.Put("/admin/tags/:id", r.tagHandler.Update)
	admin.Delete("/admin/tags/:id", r.tagHandler.Delete)
	admin.Get("/admin/comments", r.commentHandler.AdminList)
	admin.Put("/admin/comments/:id/status", r.commentHandler.UpdateStatus)
	admin.Delete("/admin/comments/:id", r.commentHandler.Delete)
	admin.Put("/admin/settings/:key", r.settingHandler.Set)
	admin.Post("/admin/settings/icon", r.settingHandler.UploadImage)
	admin.Post("/admin/upload/image", r.settingHandler.UploadImage)
	admin.Get("/admin/themes", r.themeHandler.List)
	admin.Post("/admin/themes/upload", r.themeHandler.Upload)
	admin.Post("/admin/themes/download", r.themeHandler.Download)
	admin.Get("/admin/themes/download/progress", r.themeHandler.DownloadProgress)
	admin.Post("/admin/themes/:id/activate", r.themeHandler.Activate)
	admin.Delete("/admin/themes/:id", r.themeHandler.Delete)
	admin.Get("/admin/themes/remote", r.themeHandler.FetchRemote)
	admin.Get("/admin/themes/:id/settings", r.themeHandler.GetSettings)
	admin.Put("/admin/themes/:id/settings", r.themeHandler.UpdateSettings)
}

type Server struct {
	app *fiber.App
}

func NewServer() *Server {
	return &Server{
		app: fiber.New(),
	}
}

func (s *Server) App() *fiber.App {
	return s.app
}

func (s *Server) Listen(addr string) error {
	return s.app.Listen(addr)
}

func (s *Server) Shutdown() error {
	return s.app.Shutdown()
}

func SetupRouter(
	app *fiber.App,
	jwtManager *utils.JWTManager,
	redisClient *utils.RedisClient,
	userHandler *handler.UserHandler,
	articleHandler *handler.ArticleHandler,
	categoryHandler *handler.CategoryHandler,
	tagHandler *handler.TagHandler,
	commentHandler *handler.CommentHandler,
	haloCommentHandler *handler.HaloCommentHandler,
	haloSearchHandler *handler.HaloSearchHandler,
	haloUpvoteHandler *handler.HaloUpvoteHandler,
	settingHandler *handler.SettingHandler,
	themeHandler *handler.ThemeHandler,
) {
	authMiddleware := middleware.NewAuthMiddleware(jwtManager, redisClient)

	router := NewRouter(
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
		authMiddleware,
	)

	router.Setup(app)
}
