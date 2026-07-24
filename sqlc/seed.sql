-- Default settings for the blog (only inserted when missing, so existing values are preserved)
INSERT INTO settings (key, value, description) VALUES
  ('site_title', '"AXCE"', '网站标题，显示在浏览器标签和首页'),
  ('site_description', '"一个使用 Go 和 React 构建的博客系统"', '网站描述，用于 SEO 和首页副标题'),
  ('site_keywords', '"博客,Go,React,Web开发"', '网站关键词，用于 SEO'),
  ('site_author', '"管理员"', '网站作者名称'),
  ('site_copyright', '"© 2026 AXCE. All rights reserved."', '网站底部版权信息'),
  ('posts_per_page', '"10"', '每页显示的文章数量'),
  ('enable_comments', '"true"', '是否启用评论功能'),
  ('require_review', '"false"', '新评论是否需要审核'),
  ('site_icon', '""', '网站图标（favicon），支持 PNG/JPG/SVG 格式')
ON CONFLICT (key) DO NOTHING;

-- Demo user (password: demo123, bcrypt hash)
INSERT INTO users (username, email, password_hash, nickname, avatar, bio, "group", status)
VALUES ('admin', 'admin@example.com', '$2a$10$N9qo8uLOickgx2ZMRZoMy.MqrzQh8Y7zH0h2LJy0jH0h2LJy0jH0', '管理员', '', '博客管理员', 'admin', 1)
ON CONFLICT (username) DO UPDATE SET nickname = EXCLUDED.nickname;

-- Demo categories
INSERT INTO categories (name, slug, description) VALUES
  ('技术', 'tech', '技术相关文章'),
  ('生活', 'life', '生活随笔')
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name;

-- Demo tags
INSERT INTO tags (name, slug) VALUES
  ('Go', 'go'),
  ('Halo', 'halo'),
  ('Web', 'web')
ON CONFLICT (slug) DO UPDATE SET name = EXCLUDED.name;

-- Demo articles
WITH admin_user AS (SELECT id FROM users WHERE username = 'admin' LIMIT 1),
     tech_cat AS (SELECT id FROM categories WHERE slug = 'tech' LIMIT 1),
     go_tag AS (SELECT id FROM tags WHERE slug = 'go' LIMIT 1),
     halo_tag AS (SELECT id FROM tags WHERE slug = 'halo' LIMIT 1)
INSERT INTO articles (title, slug, summary, content, cover_url, user_id, category_id, view_count)
SELECT 'Hello Halo', 'hello-halo', '第一篇示例文章', '<p>欢迎使用 AXCE 博客系统！</p>', '', admin_user.id, tech_cat.id, 42
FROM admin_user, tech_cat
WHERE NOT EXISTS (SELECT 1 FROM articles WHERE slug = 'hello-halo');

WITH admin_user AS (SELECT id FROM users WHERE username = 'admin' LIMIT 1),
     life_cat AS (SELECT id FROM categories WHERE slug = 'life' LIMIT 1),
     web_tag AS (SELECT id FROM tags WHERE slug = 'web' LIMIT 1)
INSERT INTO articles (title, slug, summary, content, cover_url, user_id, category_id, view_count)
SELECT '生活随笔', 'life-notes', '记录生活中的点滴', '<p>这是一篇生活随笔。</p>', '', admin_user.id, life_cat.id, 18
FROM admin_user, life_cat
WHERE NOT EXISTS (SELECT 1 FROM articles WHERE slug = 'life-notes');

-- Link articles to tags
INSERT INTO article_tags (article_id, tag_id)
SELECT a.id, t.id FROM articles a, tags t WHERE a.slug = 'hello-halo' AND t.slug = 'halo'
ON CONFLICT DO NOTHING;

INSERT INTO article_tags (article_id, tag_id)
SELECT a.id, t.id FROM articles a, tags t WHERE a.slug = 'hello-halo' AND t.slug = 'go'
ON CONFLICT DO NOTHING;

INSERT INTO article_tags (article_id, tag_id)
SELECT a.id, t.id FROM articles a, tags t WHERE a.slug = 'life-notes' AND t.slug = 'web'
ON CONFLICT DO NOTHING;

-- Demo primary menu
INSERT INTO menus (name) VALUES ('primary')
ON CONFLICT (name) DO NOTHING;

WITH primary_menu AS (SELECT id FROM menus WHERE name = 'primary')
INSERT INTO menu_items (menu_id, name, url, parent_id, priority)
SELECT primary_menu.id, '首页', '/', NULL, 0 FROM primary_menu
WHERE NOT EXISTS (SELECT 1 FROM menu_items WHERE menu_id = primary_menu.id AND url = '/');

WITH primary_menu AS (SELECT id FROM menus WHERE name = 'primary'),
     archives_item AS (SELECT id FROM menu_items WHERE menu_id = (SELECT id FROM menus WHERE name = 'primary') AND url = '/archives' LIMIT 1)
INSERT INTO menu_items (menu_id, name, url, parent_id, priority)
SELECT primary_menu.id, '归档', '/archives', NULL, 1 FROM primary_menu
WHERE NOT EXISTS (SELECT 1 FROM menu_items WHERE menu_id = primary_menu.id AND url = '/archives');

WITH primary_menu AS (SELECT id FROM menus WHERE name = 'primary'),
     tags_item AS (SELECT id FROM menu_items WHERE menu_id = (SELECT id FROM menus WHERE name = 'primary') AND url = '/tags' LIMIT 1)
INSERT INTO menu_items (menu_id, name, url, parent_id, priority)
SELECT primary_menu.id, '标签', '/tags', NULL, 2 FROM primary_menu
WHERE NOT EXISTS (SELECT 1 FROM menu_items WHERE menu_id = primary_menu.id AND url = '/tags');
