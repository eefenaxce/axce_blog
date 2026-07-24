package handler

import (
	"github.com/gofiber/fiber/v3"
)

// HaloSearchHandler exposes Halo 2.x-compatible search widget endpoints.
type HaloSearchHandler struct{}

func NewHaloSearchHandler() *HaloSearchHandler {
	return &HaloSearchHandler{}
}

// Render returns the HTML for the Halo search popup modal.
// The theme's SearchWidget.open() loads this page into a modal overlay.
func (h *HaloSearchHandler) Render(c fiber.Ctx) error {
	c.Set("Content-Type", "text/html; charset=utf-8")
	return c.SendString(searchWidgetHTML)
}

const searchWidgetHTML = `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1" />
  <title>搜索</title>
  <style>
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: var(--halo-search-widget-base-font-family, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif);
      color: var(--halo-search-widget-base-color, #333);
      background: transparent;
    }
    .hsw-box {
      width: 100%;
      height: 100%;
      background: var(--halo-search-widget-base-bg-color, #fff);
      border-radius: 12px;
      overflow: hidden;
      display: flex;
      flex-direction: column;
    }
    .hsw-input-wrap {
      display: flex;
      align-items: center;
      padding: 16px 20px;
      border-bottom: 1px solid var(--halo-search-widget-base-border-color, #eee);
    }
    .hsw-input-wrap svg {
      width: 20px;
      height: 20px;
      color: #999;
      margin-right: 12px;
      flex-shrink: 0;
    }
    input {
      flex: 1;
      border: none;
      outline: none;
      font-size: 16px;
      background: transparent;
      color: var(--halo-search-widget-base-color, #333);
    }
    .hsw-results {
      flex: 1;
      overflow-y: auto;
      padding: 8px 0;
    }
    .hsw-result-item {
      display: block;
      padding: 12px 20px;
      text-decoration: none;
      color: inherit;
      border-bottom: 1px solid var(--halo-search-widget-base-border-color, #f5f5f5);
    }
    .hsw-result-item:hover {
      background: var(--halo-search-widget-base-hover-bg-color, #f5f5f5);
    }
    .hsw-result-title {
      font-weight: 600;
      margin-bottom: 4px;
      color: var(--halo-search-widget-base-color, #333);
    }
    .hsw-result-summary {
      font-size: 13px;
      color: #666;
      line-height: 1.5;
    }
    .hsw-empty, .hsw-loading {
      padding: 32px 20px;
      text-align: center;
      color: #999;
    }
    .hsw-footer {
      padding: 10px 20px;
      font-size: 12px;
      color: #999;
      border-top: 1px solid var(--halo-search-widget-base-border-color, #eee);
      text-align: center;
    }
  </style>
</head>
<body>
  <div class="hsw-box">
    <div class="hsw-input-wrap">
      <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="11" cy="11" r="8"></circle><path d="m21 21-4.3-4.3"></path></svg>
      <input type="text" id="hsw-input" placeholder="搜索文章..." autofocus />
    </div>
    <div class="hsw-results" id="hsw-results"><div class="hsw-empty">输入关键词开始搜索</div></div>
    <div class="hsw-footer">按 Enter 搜索，Esc 关闭</div>
  </div>
  <script>
    (function () {
      function escapeHtml(str) {
        if (str == null) return '';
        return String(str)
          .replace(/&/g, '&amp;')
          .replace(/</g, '&lt;')
          .replace(/>/g, '&gt;')
          .replace(/"/g, '&quot;')
          .replace(/'/g, '&#39;');
      }
      function trimText(text, max) {
        if (!text) return '';
        text = text.replace(/<[^>]+>/g, ' ').replace(/\s+/g, ' ').trim();
        return text.length > max ? text.slice(0, max) + '...' : text;
      }
      function renderItems(items) {
        const parts = [];
        for (let i = 0; i < items.length; i++) {
          const post = items[i];
          const slug = encodeURIComponent(post.slug || '');
          const title = escapeHtml(post.title || '');
          const summary = escapeHtml(trimText(post.summary || post.content, 120));
          parts.push('<a class="hsw-result-item" href="/archives/' + slug + '" target="_top">'
            + '<div class="hsw-result-title">' + title + '</div>'
            + '<div class="hsw-result-summary">' + summary + '</div>'
            + '</a>');
        }
        return parts.join('');
      }
      function doSearch() {
        const keyword = input.value.trim();
        const results = document.getElementById('hsw-results');
        if (!keyword) {
          results.innerHTML = '<div class="hsw-empty">输入关键词开始搜索</div>';
          return;
        }
        results.innerHTML = '<div class="hsw-loading">搜索中...</div>';
        fetch('/api/v1/search?keyword=' + encodeURIComponent(keyword))
          .then((r) => r.json())
          .then((data) => {
            const payload = data.data || data;
            const items = payload.items || [];
            if (!items.length) {
              results.innerHTML = '<div class="hsw-empty">未找到相关文章</div>';
              return;
            }
            results.innerHTML = renderItems(items);
          })
          .catch((e) => {
            results.innerHTML = '<div class="hsw-empty">搜索失败：' + escapeHtml(e.message) + '</div>';
          });
      }
      const input = document.getElementById('hsw-input');
      let timer = null;
      input.addEventListener('input', () => {
        clearTimeout(timer);
        timer = setTimeout(doSearch, 300);
      });
      input.addEventListener('keydown', (e) => {
        if (e.key === 'Enter') {
          clearTimeout(timer);
          doSearch();
        }
        if (e.key === 'Escape' && parent.__haloCloseSearchWidget) {
          parent.__haloCloseSearchWidget();
        }
      });
      input.focus();
      input.select();
    })();
  </script>
</body>
</html>`
