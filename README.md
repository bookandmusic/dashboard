# 导航面板系统

YAML 驱动的内网/外网双链路服务导航面板。前端 Vue 3 构建产物通过 `//go:embed` 嵌入 Go 二进制，单文件部署；修改配置自动热重载，无需重启。

## 特性

- **单二进制部署**：前端静态文件编译时嵌入 Go 二进制，运行只需 `dashboard` + 配置文件
- **配置热重载**：`fsnotify` 监听配置文件，编辑保存后自动生效（兼容编辑器 rename 式保存）
- **双主题**：`dark` 机架工业风 / `orbital` 星轨调度台（全屏 Canvas 物理轨道），配置指定或运行时按 `T` 切换
- **双链路导航**：每个站点支持内网/外网多地址，一键切换，行内地址标签直达
- **图标系统**：逐站点配置 Lucide 图标名或图片（在线链接 / `/icons/` 相对路径），缺省回退首字母
- **系统指标**：实时展示 CPU / 内存 / 磁盘 / 网络 / 运行时长，直接读 `/proc`，零外部依赖

## 快速开始

### 本地构建运行

```bash
./build.sh          # 前端构建 → go:embed → 编译单二进制
./dashboard         # 默认读取 ./config.yaml，监听 :8080
```

访问 http://localhost:8080

### Docker 部署

```bash
docker compose up -d --build
```

部署时用自己的配置覆盖 `config.yaml`；配置文件与 `/proc`、`/sys` 通过卷挂载进容器，修改配置即时生效。

## 配置

仓库内 `config.yaml` 是**示例文件**（字段说明见文件内注释），按自己环境直接修改即可，或用 `CONFIG_PATH` 指定自有配置文件。结构为 面板 → 分组 → 站点 → 地址：

```yaml
title: NAS 服务导航
theme: dark                  # dark / orbital
networkMode: intranet        # 默认网络模式 intranet / internet
groups:
  - code: DEV
    name: 研发工具
    sites:
      - name: Gitea
        desc: Git 代码托管
        icon: git-branch     # Lucide 图标名，或图片链接 / /icons/xxx.svg
        addresses:
          - { net: intranet, label: 内网, url: "http://192.168.1.69:30200" }
          - { net: internet, label: 外网, url: "https://git.example.com" }
```

相对路径图标固定从**配置文件同级的 `icons/` 目录**读取（经 `/icons/` 路由提供）。完整字段说明见 [DESIGN.md](DESIGN.md)。

## 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `CONFIG_PATH` | `config.yaml` | 配置文件路径 |
| `LISTEN_ADDR` | `:8080` | 监听地址 |
| `HOST_PROC` | `/proc` | 宿主机 proc 路径（容器内挂载后用） |

## API

- `GET /api/config` — 当前生效的导航配置
- `GET /api/stats` — 系统指标快照（10s 采样周期）

## 开发

```bash
# 后端
cd backend && go test ./... && go run .

# 前端（Vite dev server，自动代理 /api 到 :8080）
cd frontend && npm install && npm run dev
```
