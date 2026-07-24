package thymeleaf

import (
	"fmt"
	"strings"
	"testing"
)

func TestURLExprInComplexExpression(t *testing.T) {
	ctx := NewContext()
	ctx.Set("__theme_base_path", "/themes/theme-Joe3")
	ctx.Set("theme", map[string]any{
		"config": map[string]any{
			"blogger": map[string]any{
				"avatar_frame": "bilibili",
			},
		},
	})

	expr := `@{/assets/frame/}+'/'+${theme.config.blogger.avatar_frame}+'.png'`
	got, err := evalStandard(expr, ctx)
	if err != nil {
		t.Fatalf("evalStandard error: %v", err)
	}
	want := "/themes/theme-Joe3/assets/frame/bilibili.png"
	if toStr(got) != want {
		t.Fatalf("got %q, want %q", toStr(got), want)
	}
}

func TestURLExprStandaloneKeepsTrailingSlash(t *testing.T) {
	ctx := NewContext()
	ctx.Set("__theme_base_path", "/themes/theme-Joe3")

	got, err := evalStandard(`@{/assets/frame/}`, ctx)
	if err != nil {
		t.Fatalf("evalStandard error: %v", err)
	}
	want := "/themes/theme-Joe3/assets/frame/"
	if toStr(got) != want {
		t.Fatalf("got %q, want %q", toStr(got), want)
	}
}

type mapTemplateLoader struct {
	templates map[string]string
}

func (l *mapTemplateLoader) Load(name string) (string, error) {
	if content, ok := l.templates[name]; ok {
		return content, nil
	}
	return "", fmt.Errorf("template %q not found", name)
}

func TestURLAttributeNoDoubleEscape(t *testing.T) {
	// Simulate a URL that already contains HTML entities (e.g. user pasted &amp; in config).
	ctx := NewContext()
	ctx.Set("url", "https://q.qlogo.cn/g?b=qq&amp;nk=2681640937&amp;s=640")

	loader := &mapTemplateLoader{
		templates: map[string]string{
			"test.html": `<img th:src="${url}" alt="avatar">`,
		},
	}
	engine := NewEngine(loader)
	out, err := engine.Render("test.html", ctx)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	// Should be single-escaped &amp; only, not &amp;amp;.
	if strings.Contains(out, "&amp;amp;") {
		t.Fatalf("double escaped output: %s", out)
	}
	if !strings.Contains(out, `src="https://q.qlogo.cn/g?b=qq&amp;nk=2681640937&amp;s=640"`) {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestJSInlineNoDoubleEscape(t *testing.T) {
	// Simulate ThemeConfig.favicon injected via JS inline syntax.
	ctx := NewContext()
	ctx.Set("favicon", "https://q.qlogo.cn/g?b=qq&amp;nk=2681640937&amp;s=640")

	loader := &mapTemplateLoader{
		templates: map[string]string{
			"test.html": `<script th:inline="javascript">const ThemeConfig = { favicon: /*[[${favicon}]]*/ '' };</script>`,
		},
	}
	engine := NewEngine(loader)
	out, err := engine.Render("test.html", ctx)
	if err != nil {
		t.Fatalf("render error: %v", err)
	}

	// JS string should contain raw &, not &amp; or &amp;amp;.
	if strings.Contains(out, `&amp;`) {
		t.Fatalf("JS string still contains HTML entity: %s", out)
	}
	if !strings.Contains(out, `favicon: "https://q.qlogo.cn/g?b=qq&nk=2681640937&s=640"`) {
		t.Fatalf("unexpected JS output: %s", out)
	}
}
