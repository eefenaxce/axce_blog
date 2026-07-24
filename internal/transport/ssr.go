package transport

import (
	"encoding/json"
	"html"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"

	"github.com/eefenaxce/axce_blog/internal/service"
)

// SSRHandler reads the SPA build index.html and injects dynamic title/meta
// by simple regex replacement — no intermediate template conversion needed.
type SSRHandler struct {
	mu             sync.RWMutex
	html           string
	settingService *service.SettingService
	themeService   *service.ThemeService

	// Pre-compiled regexes for the tags we want to replace.
	titleRE      *regexp.Regexp
	descRE       *regexp.Regexp
	kwRE         *regexp.Regexp
	authorRE     *regexp.Regexp
	copyrightRE  *regexp.Regexp
	headEndRE    *regexp.Regexp
}

func NewSSRHandler(settingService *service.SettingService, themeService *service.ThemeService) *SSRHandler {
	return &SSRHandler{
		settingService: settingService,
		themeService:   themeService,
		titleRE:        regexp.MustCompile(`<title>[^<]*</title>`),
		descRE:         regexp.MustCompile(`<meta\s+name="description"\s+content="[^"]*"`),
		kwRE:           regexp.MustCompile(`<meta\s+name="keywords"\s+content="[^"]*"`),
		authorRE:       regexp.MustCompile(`<meta\s+name="author"\s+content="[^"]*"`),
		copyrightRE:    regexp.MustCompile(`<meta\s+name="copyright"\s+content="[^"]*"`),
		headEndRE:      regexp.MustCompile(`</head>`),
	}
}

// Load reads the SPA build output once at startup.
func (h *SSRHandler) Load(filepath string) error {
	content, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	h.mu.Lock()
	h.html = string(content)
	h.mu.Unlock()

	return nil
}

// Serve implements the Fiber handler. On each request it copies the base
// HTML, replaces title/meta with DB-backed values, injects active theme info,
// and serves the result.
func (h *SSRHandler) Serve(ctx fiber.Ctx) error {
	h.mu.RLock()
	base := h.html
	h.mu.RUnlock()

	title, desc, kw, author, copyright := h.loadMeta(ctx)
	activeThemeJSON := h.loadActiveTheme(ctx)

	out := h.titleRE.ReplaceAllLiteralString(base, "<title>"+html.EscapeString(title)+"</title>")
	out = h.descRE.ReplaceAllLiteralString(out, `<meta name="description" content="`+html.EscapeString(desc)+`"`)
	out = h.kwRE.ReplaceAllLiteralString(out, `<meta name="keywords" content="`+html.EscapeString(kw)+`"`)
	out = h.authorRE.ReplaceAllLiteralString(out, `<meta name="author" content="`+html.EscapeString(author)+`"`)
	out = h.copyrightRE.ReplaceAllLiteralString(out, `<meta name="copyright" content="`+html.EscapeString(copyright)+`"`)

	// 在 </head> 之前注入激活主题信息
	if activeThemeJSON != "" {
		out = h.headEndRE.ReplaceAllLiteralString(out,
			`<script>window.__ACTIVE_THEME__ = `+activeThemeJSON+`;</script></head>`)
	}

	ctx.Set("Content-Type", "text/html; charset=utf-8")
	return ctx.SendString(out)
}

// loadActiveTheme 获取当前激活主题的信息作为 JSON
func (h *SSRHandler) loadActiveTheme(ctx fiber.Ctx) string {
	if h.themeService == nil {
		return ""
	}
	theme, err := h.themeService.GetActiveTheme(ctx.Context())
	if err != nil || theme == nil {
		return ""
	}
	data, err := json.Marshal(theme)
	if err != nil {
		return ""
	}
	return string(data)
}

// loadMeta fetches settings from DB.
func (h *SSRHandler) loadMeta(ctx fiber.Ctx) (title, desc, kw, author, copyright string) {
	if h.settingService == nil {
		return
	}

	appCtx := ctx.Context()
	if v, err := h.settingService.Get(appCtx, "site_title"); err == nil && v != "" {
		title = cleanSetting(v)
	}
	if v, err := h.settingService.Get(appCtx, "site_description"); err == nil && v != "" {
		desc = cleanSetting(v)
	}
	if v, err := h.settingService.Get(appCtx, "site_keywords"); err == nil && v != "" {
		kw = cleanSetting(v)
	}
	if v, err := h.settingService.Get(appCtx, "site_author"); err == nil && v != "" {
		author = cleanSetting(v)
	}
	if v, err := h.settingService.Get(appCtx, "site_copyright"); err == nil && v != "" {
		copyright = cleanSetting(v)
	}
	return
}

func cleanSetting(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			value = value[1 : len(value)-1]
		}
	}
	return value
}
