# RemoteRun Client

一个基于 `Vite + Vue 3 + TypeScript` 的远程命令控制台。

## 特性

- 使用最新稳定版 `Vite` 与 `Vue`
- 服务器配置放在项目根目录 `data/`
- 默认自带 `sample.json` 作为模板，运行时会自动忽略
- 前端不直接读取密码，也不直接访问远端服务器
- 通过 Vite 中间层读取本地配置并转发命令，规避 CORS 并减少敏感信息暴露

## 安装依赖

```bash
npm_config_cache=/tmp/remoterun-client-npm-cache npm install
```

## 启动开发环境

```bash
npm run dev
```

## 构建

```bash
npm run build
```

## 预览

```bash
npm run preview
```

`preview` 模式同样保留了读取 `data/` 目录和代理远端命令的能力，适合本地使用。

## 数据目录格式

在 `data/` 目录新增任意 `.json` 文件即可，文件名会作为服务器 ID。

示例：

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
- `password` 只保留在本地服务端中间层，浏览器拿不到
- 页面上只显示命令别名，不暴露具体命令内容

## 对接后端

项目按 `/Users/nocilantro/remoterun-server` 的接口约定转发请求：

- 目标接口：`POST http://host:port/run`
- 请求体：`{ "command": "...", "password": "..." }`
- 响应体：兼容 `success / exit_code / stdout / stderr / combined_log`

## 生产环境测试使用

```bash
npm run dev -- --host 0.0.0.0 --port 5173
```