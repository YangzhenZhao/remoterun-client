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
- 服务器配置仍保存在项目根目录 `data/`
- 浏览器只看到命令别名，不会拿到远端密码和真实命令内容

## 项目结构

```text
.
├── backend/        # Gin + PostgreSQL 后端
├── data/           # 服务器 JSON 配置
└── src/            # Vue 前端
```

## 数据目录格式

在 `data/` 目录新增任意 `.json` 文件即可，文件名会作为服务器 ID。

```json
{
  "alias": "生产环境",
  "host": "10.0.0.8",
  "port": 8080,
  "password": "your-password",
  "commands": [
    {
      "alias": "查看版本",
      "command": "cat /opt/app/version.txt"
    },
    {
      "alias": "重启服务",
      "command": "systemctl restart my-app"
    }
  ]
}
```

注意：

- `sample.json` 永远不会出现在页面中
- `password` 只保留在后端内存和请求转发阶段，浏览器拿不到
- 页面上只显示命令别名，不暴露具体命令内容

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

按你的环境修改 `.env`，关键变量如下：

```bash
APP_ADDR=:8080
ALLOWED_ORIGIN=http://localhost:5173
COOKIE_SECURE=false
DATA_DIR=../data
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

后端会自动创建 `users` 表。

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
