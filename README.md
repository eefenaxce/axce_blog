# Axce Blog

一个基于 Go + React Router 构建的类 Halo 博客系统，支持主题渲染、文章管理、评论、分类/标签、用户认证等完整功能。

## 技术栈

- **后端**：Go 1.26 + Fiber v3 + PostgreSQL + Redis
- **前端**：React Router v7 + React 19 + Tailwind CSS v4 + shadcn/ui
- **模板引擎**：thymeleaf-go（兼容 Halo 2.x 主题）
- **部署**：Docker / Docker Compose + GitHub Actions CI/CD

## 功能特性

- 文章/页面发布、编辑与管理
- Markdown / 富文本内容支持
- 分类与标签管理
- 评论系统（支持审核、嵌套回复）
- 用户注册、登录、密码重置
- 基于 Halo 2.x 主题引擎的前端渲染
- 主题下载、上传、切换与配置
- 站点设置（图标、SEO、评论策略等）
- 后台管理面板
- Docker 一键部署

## 快速开始

### 环境要求

- Go 1.26+
- Node.js 24+
- pnpm
- PostgreSQL 14+
- Redis 7+（可选，但推荐）

### 本地开发

```bash
# 1. 克隆项目
git clone <your-repo-url>
cd axce_blog

# 2. 复制并编辑配置文件
cp config.example.yaml config.yaml

# 3. 启动后端
go run ./cmd/server

# 4. 启动前端（新终端）
cd web
pnpm install
pnpm dev
```

访问：

- 前台首页：[http://localhost:8080](http://localhost:8080)
- 后台管理：[http://localhost:8080/admin](http://localhost:8080/admin)
- 前端开发服务器：[http://localhost:5173](http://localhost:5173)

## Docker 部署

### 1. 准备配置

```bash
cp config.example.yaml config.yaml
# 编辑 config.yaml，将 database/redis 主机改为服务名
```

### 2. 启动

```bash
docker compose up --build -d
```

访问：[http://localhost:8080](http://localhost:8080)

### 3. 停止

```bash
docker compose down
```

更多部署方式见 [CI/CD 自动构建](#cicd-自动构建)。

## 目录结构

```
.
├── cmd/server           # Go 后端入口
├── internal
│   ├── config           # 配置加载
│   ├── db               # 数据库与 Redis 连接
│   ├── models           # 数据模型
│   ├── repository       # 数据访问层
│   ├── service          # 业务逻辑层
│   ├── transport        # HTTP 路由与处理器
│   └── utils            # 工具函数
├── pkg/thymeleaf-go     # 模板引擎
├── sqlc                 # 数据库 schema、迁移与查询
├── web                  # React Router 前端
├── Dockerfile           # 镜像构建
├── docker-compose.yml   # 本地/服务器一键部署
└── deploy.sh            # 服务器拉取运行脚本
```

## CI/CD 自动构建

项目已配置 GitHub Actions 工作流，推送代码到 `main` 分支或 `v*` tag 时会自动构建并推送 Docker 镜像到 Docker Hub。

### 配置 Secrets

在 GitHub 仓库 Settings → Secrets and variables → Actions 中添加：

| Secret | 说明 |
| --- | --- |
| `DOCKER_USERNAME` | Docker Hub 用户名 |
| `DOCKER_PASSWORD` | Docker Hub 密码或 Access Token |

### 服务器部署

```bash
# 修改 deploy.sh 中的镜像名
IMAGE="your-dockerhub-username/axce-blog:latest"

# 执行部署
chmod +x deploy.sh
./deploy.sh
```

## 配置说明

配置文件为 `config.yaml`，主要字段：

| 节点 | 说明 |
| --- | --- |
| `server.port` | 服务端口，默认 `8080` |
| `database.*` | PostgreSQL 连接信息 |
| `redis.*` | Redis 连接信息（可选） |
| `jwt.secret` | JWT 签名密钥，至少 32 位 |
| `email.*` | SMTP 邮件发送配置（用于注册/找回密码） |

## 许可证

本项目基于 [MIT License](LICENSE) 开源。