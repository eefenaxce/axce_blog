package thymeleaf

import (
	"strings"
	"testing"
)

type memoryLoader struct {
	templates map[string]string
}

func (m *memoryLoader) Load(name string) (string, error) {
	if content, ok := m.templates[name]; ok {
		return content, nil
	}
	loader := NewFileSystemLoader("../../themes/theme-Joe3-1.5.0/templates")
	return loader.Load(name)
}

func TestArrayIndexExpression(t *testing.T) {
	memLoader := &memoryLoader{
		templates: map[string]string{
			"test.html": `<html><body>
<span class="direct">[[${post.status.contributors[0]}]]</span>
<span class="size">[[${post.status.contributors.size()}]]</span>
<span class="last">[[${post.status.contributors[post.status.contributors.size()-1]}]]</span>
</body></html>`,
		},
	}
	engine := NewEngine(memLoader)
	ctx := NewContext()
	ctx.Set("post", map[string]any{
		"status": map[string]any{
			"contributors": []string{"测试作者"},
		},
	})

	html, err := engine.Render("test.html", ctx)
	if err != nil {
		t.Fatalf("render template failed: %v", err)
	}
	t.Logf("output:\n%s", html)
	if !strings.Contains(html, `<span class="direct">测试作者</span>`) {
		t.Errorf("direct index failed; got:\n%s", html)
	}
	if !strings.Contains(html, `<span class="size">1</span>`) {
		t.Errorf("size() failed; got:\n%s", html)
	}
	if !strings.Contains(html, `<span class="last">测试作者</span>`) {
		t.Errorf("dynamic last index failed; got:\n%s", html)
	}
}

func TestPostCopyrightWithArrayIndex(t *testing.T) {
	memLoader := &memoryLoader{
		templates: map[string]string{
			"test-post": `<html><body>
<th:block th:with="contributor = ${contributorFinder.getContributor(post.status.contributors[post.status.contributors.size()-1])}">
  <span class="inline">[[${contributor.displayName}]]</span>
  <span class="attr" th:text="${contributor.displayName}"></span>
</th:block>
</body></html>`,
		},
	}
	engine := NewEngine(memLoader)
	ctx := NewContext()
	ctx.Set("post", map[string]any{
		"status": map[string]any{
			"contributors": []string{"测试作者"},
		},
	})
	ctx.Set("contributorFinder", &mockContributorFinder{})

	html, err := engine.Render("test-post", ctx)
	if err != nil {
		t.Fatalf("render template failed: %v", err)
	}
	t.Logf("output:\n%s", html)
	if !strings.Contains(html, "测试作者") {
		t.Errorf("expected 测试作者 in output, got:\n%s", html)
	}
}

type mockContributorFinder struct{}

func (m *mockContributorFinder) GetContributor(name string) map[string]any {
	return map[string]any{
		"name":        name,
		"displayName": name,
		"avatar":      "",
		"bio":         "",
	}
}
