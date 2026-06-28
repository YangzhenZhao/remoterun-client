# RemoteRun

一个包含前后端的远程命令控制台：

- 前端：`Vite + Vue 3 + TypeScript`
- 后端：`Go + Gin`
- 数据库：`PostgreSQL`
- 认证：`Session + HttpOnly Cookie`
- 安全：`CSRF Token + 安全响应头`

## 当前能力

- 账号密码登录后才可访问服务器列表和执行命令
- 用户数据存放在 `PostgreSQL`
- 会话通过 `HttpOnly Cookie` 保存，前端脚本无法直接读取
- 关键写操作要求携带 `X-CSRF-Token`，并校验请求来源
- 服务端返回 `CSP / X-Frame-Options / X-Content-Type-Options` 等安全响应头
- 服务器和命令配置存放在 `PostgreSQL`
- 浏览器只看到命令别名，不会拿到远端密码和真实命令内容

## 项目结构

```text
.
├── backend/        # Gin + PostgreSQL 后端
├── nginx/          # Docker 部署时的前端反向代理配置
└── src/            # Vue 前端
```

## Docker 一键部署

适合直接启动完整环境，包括：

- `frontend`：构建后的 Vue 静态站点，由 `nginx` 提供服务
- `backend`：Go API 服务
- `db`：PostgreSQL

先复制 Docker 环境变量模板：

```bash
cp .env.example .env
```

按需修改根目录 `.env`，至少建议替换：

```bash
SESSION_SECRET=change-this-session-secret-please-32chars
ADMIN_PASSWORD=change-this-password
APP_ORIGIN=http://localhost:8081
FRONTEND_PORT=8081
```

启动完整环境：

```bash
# Docker Compose V2
docker compose up -d --build

# 如果你的环境只支持旧版命令
docker-compose up -d --build
```

访问地址：

- 前端：`http://localhost:8081`
- 后端 API：由前端同源反代到 `/api`

常用命令：

```bash
# 查看日志
docker compose logs -f
docker-compose logs -f

# 停止服务
docker compose down
docker-compose down

# 连数据库数据一起清理
docker compose down -v
docker-compose down -v
```

说明：

- 首次启动会自动初始化数据库，并根据 `.env` 中的 `ADMIN_USERNAME / ADMIN_PASSWORD` 创建或更新管理员账号
- 生产环境建议将 `COOKIE_SECURE=true`，并把 `APP_ORIGIN` 改成真实 HTTPS 域名

## Docker 本地测试

如果你想在容器里跑开发环境，使用下面的开发编排：

```bash
cp .env.example .env

# Docker Compose V2
docker compose -f docker-compose.dev.yml up --build

# 如果你的环境只支持旧版命令
docker-compose -f docker-compose.dev.yml up -d --build 
```

停止:  

```bash
docker-compose -f docker-compose.dev.yml down
```

如果出现 `unknown shorthand flag: 'f' in -f` 这类报错，通常表示当前 Docker 环境没有启用 Compose V2，请改用 `docker-compose`。

访问地址：

- 前端开发服务器：`http://localhost:5173`
- 后端 API：`http://localhost:8080`
- PostgreSQL：`localhost:5432`

这一模式下：

- 前端运行 `Vite` 开发服务器
- 后端运行 `go run ./cmd/server`
- 项目源码以卷挂载到容器，适合本地联调和接口测试

## 非 Docker 启动

如果你不使用 Docker，仍然可以按下面方式分别启动前后端。

## 服务器数据

服务器信息和命令配置已存放到数据库中，不再依赖项目根目录 `data/` 下的 JSON 文件。

当前推荐方式：

- 登录后直接在前端“服务器列表”页手动新增服务器
- 后端把服务器和命令保存到 `servers / server_commands` 两张表
- 浏览器只看到命令别名，不会拿到密码和真实命令内容

## 启动 PostgreSQL

先准备一个数据库，例如：

```sql
CREATE DATABASE remoterun;
```

## 启动后端

复制环境变量模板：

```bash
cd backend
cp .env.example .env
```

按你的环境修改 `backend/.env`，关键变量如下：

```bash
APP_ADDR=:8080
ALLOWED_ORIGIN=http://localhost:5173
COOKIE_SECURE=false
DATABASE_URL=postgres://postgres:postgres@localhost:5432/remoterun?sslmode=disable
SESSION_NAME=remoterun_session
SESSION_SECRET=replace-with-a-random-32-char-secret
UPSTREAM_TIMEOUT=60s
ADMIN_USERNAME=admin
ADMIN_PASSWORD=change-this-password
```

说明：

- `SESSION_SECRET` 至少 32 个字符
- `ADMIN_USERNAME` 和 `ADMIN_PASSWORD` 会在启动时自动创建或更新一个管理员账号
- 生产环境建议把 `COOKIE_SECURE=true`，并通过 HTTPS 提供服务

启动后端：

```bash
cd backend
set -a
source .env
set +a
go run ./cmd/server
```

后端会自动创建 `users / servers / server_commands` 表。

## 启动前端

安装依赖：

```bash
npm_config_cache=/tmp/remoterun-client-npm-cache npm install
```

开发环境默认把 `/api` 代理到 `http://localhost:8080`。如果后端地址不同，可设置：

```bash
export VITE_API_TARGET=http://localhost:8080
```

启动前端：

```bash
npm run dev
```

## 构建前端

```bash
npm run build
```

## 后端构建

```bash
cd backend
go build ./...
```

## 接口说明

认证接口：

- `GET /api/auth/session`
- `POST /api/auth/login`
- `POST /api/auth/logout`

业务接口：

- `GET /api/servers`
- `GET /api/servers/:id`
- `POST /api/servers`
- `POST /api/run`

远端命令执行仍按下列约定转发到 remoterun-server：

- 目标接口：`POST http://host:port/run`
- 请求体：`{ "command": "...", "password": "..." }`
- 响应体：兼容 `success / exit_code / stdout / stderr / combined_log`

## 安全说明

- `Session Cookie` 设置为 `HttpOnly`
- 写接口要求 `X-CSRF-Token`
- 后端校验 `Origin`
- 前端不使用 `v-html`，渲染走 Vue 默认转义，降低 XSS 风险
- 后端统一返回安全响应头，降低点击劫持和内容嗅探风险

### 国内镜像使用

修改 ~/.colima/default/colima.yaml

```yaml
network:
  address: false
  mode: shared
  interface: en0
  preferredRoute: false
  dns:
    - 223.5.5.5
    - 119.29.29.29
  dnsHosts: {}
  hostAddresses: false
  gatewayAddress: 192.168.5.2

docker:
  registry-mirrors:
    - https://docker.m.daocloud.io
    - https://mirror.ccs.tencentyun.com
    - https://hub-mirror.c.163.com
```
