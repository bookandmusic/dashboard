# 导航面板系统 设计文档

## 1. 需求概述

构建一个导航面板系统，满足以下核心需求：

- 前端以静态文件方式部署（Vue 3 构建产物）
- 前端静态文件通过 `//go:embed` 打包到 Go 二进制中，单文件部署
- 导航结构由 YAML 配置文件驱动，运行时读取
- 修改配置文件后自动热重载，无需重启服务
- 后端提供配置 API，前端渲染导航 UI
- 后端从 `/proc`、`/sys` 读取系统指标（CPU、内存、磁盘、网络、运行时长），通过 `/api/stats` 提供给前端展示
- 支持 Docker 部署，通过挂载宿主机 `/proc`、`/sys` 和 `config.yaml` 实现容器内运行

---

## 2. 系统架构

```
┌─────────────────────────────────────────┐
│              浏览器                       │
│  ┌───────────────────────────────────┐  │
│  │   Vue 3 SPA (静态文件)              │  │
│  │   ┌─────────────────────────┐     │  │
│  │   │  NavPanel 组件           │     │  │
│  │   │  - 读取 /api/config      │     │  │
│  │   │  - 读取 /api/stats       │     │  │
│  │   │  - 定时轮询 (10s)        │     │  │
│  │   │  - 渲染导航树 + 系统指标  │     │  │
│  │   └─────────────────────────┘     │  │
│  └───────────────────────────────────┘  │
│                    ▲                     │
│                    │ HTTP GET            │
└────────────────────┼─────────────────────┘
                     │
┌────────────────────┼─────────────────────┐
│           Go HTTP 服务 (端口 8080)        │
│  ┌─────────────────┴─────────────┐       │
│  │  /api/config                   │       │
│  │  - 返回当前生效的导航配置 (JSON) │       │
│  └───────────────────────────────┘       │
│  ┌───────────────────────────────┐       │
│  │  /api/stats                    │       │
│  │  - 返回系统指标 (JSON)         │       │
│  │  - 内存缓存，10s 采样周期      │       │
│  └───────────────────────────────┘       │
│  ┌───────────────────────────────┐       │
│  │  / → 静态文件服务              │       │
│  │  - 通过 //go:embed 嵌入二进制   │       │
│  │  - SPA fallback → index.html  │       │
│  └───────────────────────────────┘       │
│  ┌───────────────────────────────┐       │
│  │  Config 模块                    │       │
│  │  - 启动时读取 config.yaml      │       │
│  │  - fsnotify 监听文件变更       │       │
│  │  - 变更后原子替换内存配置       │       │
│  │  - 线程安全 (sync.RWMutex)     │       │
│  └───────────────────────────────┘       │
│  ┌───────────────────────────────┐       │
│  │  Stats 模块                    │       │
│  │  - 读取 /proc/stat, meminfo   │       │
│  │  - 读取 /proc/net/dev, uptime │       │
│  │  - syscall.Statfs 读磁盘      │       │
│  │  - 内存缓存，定时采样          │       │
│  └───────────────────────────────┘       │
└─────────────────────────────────────────┘
                     │
          ┌──────────┼──────────┐
          │          │          │
   ┌──────┴──────┐ ┌─┴──┐  ┌───┴──┐
   │ config.yaml │ │/proc│  │ /sys │
   └─────────────┘ └────┘  └──────┘
    编辑即生效      系统指标来源（容器内挂载宿主机路径）
```

---

## 3. 前端设计（Vue 3 + Vite）

### 技术选型

| 项 | 选择 | 理由 |
|---|------|------|
| 框架 | Vue 3 (Composition API + `<script setup>`) | 轻量，组合式 API 适合仪表盘类应用 |
| 构建工具 | Vite | 快速 HMR，构建产物为纯静态文件 |
| 路由 | 无（纯导航面板，不做页面路由） | YAGNI，当前需求只展示导航结构 |
| HTTP 客户端 | 原生 fetch | 只调用一个 API，无需引入 axios |
| 图标 | lucide-vue-next | 统一 Lucide 线性图标；config 可逐站点指定图标名或图片 |
| 样式 | 原生 CSS（双主题） | 不引入 UI 框架，dark / orbital 两套独立样式 |

### 视觉参考

内置两套可切换主题（`config.yaml` 的 `theme` 字段 + 运行时 `T` 键）：

- **dark（机架工业风）**：暗色可读性优化，分组可折叠 + 行可展开，大号 display 站名 + 强调色条 hover，控制条集成搜索 + 内外网拨杆，右侧监视器展示真实系统指标
- **orbital（星轨调度台）**：浅色暖底全屏 Canvas 物理轨道系统——NAS 核心为鎏金太阳，每个分组一条轨道环、每个站点一颗图标行星；磁吸抓握 + 冻结瞄准、拖拽旋转（惯性动量）、滚轮切环，HUD 四角仪表（遥测接真实 `/api/stats`）、详情舱、全屏检索、开机自检

### 组件树

```
App.vue                        ← 数据编排 + 主题调度（config.theme / T 键 / localStorage）
 ├── themes/DarkTheme.vue      ← dark 主题骨架 + 快捷键（/ 检索、ESC）
 │    ├── TrafficCanvas.vue    ← 背景流量动画
 │    ├── BootScreen.vue       ← BIOS 开机自检
 │    ├── StatusBar.vue        ← 顶部状态条（品牌 + 状态指示 + 时钟）
 │    ├── Masthead.vue         ← 首屏大标题 + 旋转徽章 + 统计
 │    ├── NavPanel.vue         ← 控制条（搜索 + 网络拨杆）+ 机架
 │    │    └── NavGroup.vue    ← 分组（可折叠）
 │    │         └── NavItem.vue← 站点行（双地址芯片 + 展开详情）
 │    ├── Rail.vue             ← 右侧监视器（真实系统指标）
 │    └── Marquee.vue          ← 跑马灯
 ├── themes/OrbitalTheme.vue   ← orbital 主题（Canvas 轨道物理 + HUD + 详情舱 + 检索层）
 └── components/SiteIcon.vue   ← 图标渲染（Lucide 名 / 图片 / 首字母回退，双主题策略）
```

### 行为定义

- 页面加载时从 `/api/config` 获取导航配置，从 `/api/stats` 获取系统指标
- **定时轮询**：每 10 秒重新请求 `/api/config` 和 `/api/stats`
- 分组 → 站点两级渲染；展开/折叠状态由组件内部控制（local state）
- 站点行点击：桌面端打开当前网络模式对应地址（缺失则回退首个）；移动端点击展开详情
- `/api/stats` 请求失败时，状态条/侧栏指标降级，不影响导航功能

### 非功能性要求

- 构建产物输出到 `frontend/dist/`
- 产物为纯静态文件（HTML + JS + CSS）
- 不支持 SSR，纯客户端渲染
- 构建产物最终通过 `//go:embed` 打包到 Go 二进制中，运行时无需携带 `dist/` 目录

---

## 4. 后端设计（Go）

### 技术选型

| 项 | 选择 | 理由 |
|---|------|------|
| 语言 | Go | 编译为单二进制，零运行时依赖；标准库 HTTP 服务足够 |
| HTTP 框架 | `net/http`（标准库） | 仅两个路由，无需引入第三方框架 |
| 静态文件嵌入 | `embed`（标准库） | Go 1.16+ 原生支持，编译时将 `frontend/dist` 打包到二进制 |
| YAML 解析 | `gopkg.in/yaml.v3` | Go 生态标准 YAML 库 |
| 文件监听 | `github.com/fsnotify/fsnotify` | 跨平台文件变更通知，无需轮询文件系统 |
| 配置线程安全 | `sync.RWMutex` | 读多写少场景，读锁不互斥 |

### 模块划分

```
backend/
 ├── main.go         服务入口：注册路由、启动 HTTP + 文件监听
 ├── config.go       配置结构体定义 + YAML 解析
 ├── watcher.go      fsnotify 监听 + 热重载逻辑
 ├── stats.go        系统指标采集（/proc 读取 + 内存缓存）
 ├── handler.go      HTTP handler（/api/config + /api/stats + 静态文件）
 └── go.mod          模块定义
```

### //go:embed 嵌入策略

```go
//go:embed frontend/dist
var staticFS embed.FS
```

- 编译时 Go 会将 `frontend/dist/` 目录下所有文件打包进二进制
- 运行时通过 `http.FS(staticFS)` 创建 `http.FileServer`
- SPA fallback 通过 `http.FileServer` 的自动 `index.html` 机制或自定义 `fs.FS` 包装实现
- `config.yaml` **不嵌入**，运行时从文件系统读取，支持热重载

### 核心流程

#### 构建流程
1. `npm run build` — Vite 将前端构建到 `frontend/dist/`
2. `go build -o dashboard ./backend` — Go 编译时通过 `//go:embed` 将 `frontend/dist/` 打包进二进制

#### 启动流程
1. 读取 `config.yaml`，解析为内存中的 `Config` 结构体
2. 启动 goroutine 运行 `fsnotify` 监听 `config.yaml`
3. 注册 HTTP 路由（用 `embed.FS` 提供静态文件服务）
4. 启动 HTTP 服务

#### 热重载流程
1. `fsnotify` 收到 `config.yaml` 的 `Write` 事件
2. 延迟 100ms（防抖，避免编辑器多次触发）
3. 重新读取并解析 YAML
4. 加写锁，原子替换内存中的配置指针
5. 释放写锁
6. 日志记录重载结果（成功/失败）
7. 如果解析失败，保持旧配置不变，日志输出错误

#### API 请求流程
1. 收到 `GET /api/config`
2. 加读锁，读取当前配置
3. 序列化为 JSON
4. 返回 `Content-Type: application/json`

### 系统指标采集（stats.go）

不依赖外部监控服务，直接从 `/proc`、`/sys` 读取，零外部依赖：

| 指标 | 来源 | 采集方式 |
|------|------|---------|
| CPU 使用率 | `/proc/stat` | 两次采样算差值 |
| 内存 | `/proc/meminfo` | `MemTotal` / `MemAvailable` |
| 磁盘 | `syscall.Statfs` | 标准库直接调用 |
| 网络流量 | `/proc/net/dev` | 两次采样算速率 |
| 系统运行时长 | `/proc/uptime` | 一个浮点数 |
| 负载 | `/proc/loadavg` | 三个浮点数 |

**路径适配**：通过环境变量 `HOST_PROC`（默认空，即 `/proc`）区分裸机与容器环境。容器内挂载宿主机 `/proc` 到 `/host/proc`，设置 `HOST_PROC=/host/proc` 即可。

**缓存策略**：后台 goroutine 每 10 秒采样一次，结果写入内存缓存（`sync.RWMutex` 保护）。`/api/stats` 请求直接读缓存，不触发实时采集。

---

## 5. YAML 配置格式

导航结构为 **面板 → 分组（group）→ 站点（site）→ 地址（address）** 四级。每个站点必须归属某个分组；每个站点支持一个或多个地址（典型为内网 + 外网两条）。

```yaml
# 导航面板配置 —— 修改此文件后自动生效，无需重启
title: 内部工具导航           # 面板标题
theme: dark                 # 主题：dark（机架工业风）/ orbital（星轨调度台）
networkMode: intranet       # 默认网络模式：intranet / internet

groups:
  - code: DEV               # 分组代号（展示用）
    name: 研发工具           # 分组名
    sites:
      - name: 代码仓库
        desc: Git 代码托管   # 可选描述
        icon: git-branch    # 可选图标：Lucide 名 或 图片（在线链接 / /icons/x.svg 相对路径）
        addresses:          # 一个或多个地址
          - net: intranet   # 地址所属网络：intranet / internet
            label: 内网      # 展示标签
            url: http://git.intra.example
          - net: internet
            label: 外网
            url: https://git.example.com

      - name: 制品仓库
        addresses:          # 只有内网地址的站点
          - net: intranet
            label: 内网
            url: http://artifacts.intra.example

  - code: OFFICE
    name: 办公协同
    sites:
      - name: 云资源
        addresses:          # 只有外网地址的站点
          - net: internet
            label: 外网
            url: https://cloud.example.com
```

### 配置项说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `title` | string | 是 | 面板标题 |
| `desc` | string | 否 | 面板描述（首屏标题下方，留空则不显示） |
| `theme` | string | 否 | 主题：`dark` / `orbital`（默认 `dark`；运行时按 `T` 切换，手动选择优先于配置） |
| `networkMode` | string | 否 | 默认网络模式：`intranet` / `internet`（默认 `intranet`） |
| `groups` | array | 是 | 分组列表 |
| `groups[].code` | string | 否 | 分组代号（展示用，如 `DEV`） |
| `groups[].name` | string | 是 | 分组名 |
| `groups[].sites` | array | 是 | 该分组下的站点列表 |
| `sites[].name` | string | 是 | 站点名 |
| `sites[].desc` | string | 否 | 站点描述 |
| `sites[].icon` | string | 否 | 图标：Lucide 图标名（如 `terminal`）或图片（`http(s)://…` 在线链接、`/icons/x.svg` 等相对路径）；相对路径固定从**配置文件同级的 `icons/` 目录**读取（经后端 `/icons/` 路由提供）；缺省或图标名无效时回退站点首字母 |
| `sites[].addresses` | array | 是 | 地址列表（≥1 个） |
| `addresses[].net` | string | 是 | 所属网络：`intranet` / `internet` |
| `addresses[].label` | string | 是 | 展示标签（如 内网 / 外网） |
| `addresses[].url` | string | 是 | 地址 URL |

### 多地址交互规则

- **全局网络模式**：面板顶部提供 内网/外网 切换，记忆选择，作为站点点击的默认地址解析依据。
- **站点行点击**：打开当前网络模式对应的地址；若该站点没有当前模式的地址，回退到其首个地址。
- **行内地址标签**：站点只展示其实际拥有的地址标签，点击标签直接打开该地址（覆盖当前模式）。
- 单地址站点不展示可选标签，以"仅内网/仅外网"标记提示。

### 样式方案（双主题）

- 两套独立样式：`frontend/src/style.css`（dark，作用域 `body.theme-dark`）与 `frontend/src/orbital.css`（orbital，作用域 `body.theme-orbital`），App 按主题挂载对应主题组件并切换 body 类名
- 主题来源优先级：用户手动切换（`T` 键，localStorage 记忆）＞ `config.yaml` 的 `theme` ＞ 默认 `dark`
- **图标渲染策略（SiteIcon.vue）**：同一 `icon` 字段按主题语言渲染——
  - dark：Lucide 名 → 分组强调色线性图标；图片 → 直接全彩（暗色中性底包容品牌色）
  - orbital：Lucide 名 → 环色相染色；图片 → `mask-image` 剪影染环色，悬停交叉淡入原始真彩
  - 未知图标名 / 缺省 → 站点首字母
- dark 可读性经过专门优化：背景层级提亮（底色 `#171f1c` → 面板逐级提亮），文字对比度达标（主文字 ≥15:1、次级 ≥7:1、三级 ≥4.5:1）
- orbital 设计参考 `demo/4.html`：全屏 Canvas 轨道物理系统、HUD 四角仪表、磁吸抓握 + 冻结瞄准、拖拽惯性旋转

---

## 6. API 设计

### GET /api/config

返回当前生效的导航配置。

响应体与 `config.yaml` 同构（YAML → JSON），结构为 面板 → 分组 → 站点 → 地址：

```json
{
  "title": "内部工具导航",
  "theme": "dark",
  "networkMode": "intranet",
  "groups": [
    {
      "code": "DEV",
      "name": "研发工具",
      "sites": [
        {
          "name": "代码仓库",
          "desc": "Git 代码托管",
          "addresses": [
            {"net": "intranet", "label": "内网", "url": "http://git.intra.example"},
            {"net": "internet", "label": "外网", "url": "https://git.example.com"}
          ]
        },
        {
          "name": "制品仓库",
          "addresses": [
            {"net": "intranet", "label": "内网", "url": "http://artifacts.intra.example"}
          ]
        }
      ]
    }
  ]
}
```

**Status Codes:**
- `200 OK` — 正常返回配置

**前端请求**：使用 axios 请求 `/api/config`。`demo/theme1/api/config.json` 即该接口的本地模拟，demo 的 `CONFIG_URL` 指向它，生产环境改为真实接口地址即可。

### GET /api/stats

返回系统指标快照（后端内存缓存，10 秒采样周期）。

```json
{
  "cpu": {
    "usage": 23.5,
    "cores": 8,
    "loadavg": [1.2, 0.8, 0.6]
  },
  "memory": {
    "total": 68719476736,
    "available": 45634027520,
    "usage": 33.6
  },
  "disks": [
    { "name": "sda1", "mount": "/volume1", "total": 3920558678016, "available": 3225974509568, "usage": 17.72 },
    { "name": "md0", "mount": "/volume2", "total": 486076882944, "available": 454631280640, "usage": 6.47 }
  ],
  "network": {
    "rx_bytes": 123456789,
    "tx_bytes": 987654321,
    "rx_rate": 1024000,
    "tx_rate": 512000
  },
  "uptime": 11059210,
  "hostname": "nas"
}
```

**字段说明：**

| 字段 | 类型 | 说明 |
|------|------|------|
| `cpu.usage` | float | CPU 使用率（%），两次采样差值计算 |
| `cpu.cores` | int | CPU 核心数 |
| `cpu.loadavg` | [3]float | 1/5/15 分钟负载 |
| `memory.total` | int | 总内存（字节） |
| `memory.available` | int | 可用内存（字节） |
| `memory.usage` | float | 内存使用率（%） |
| `disks[].name` | string | 磁盘名（设备名清理后，如 `sda1`、`md0`、LVM 卷名） |
| `disks[].mount` | string | 挂载点（按设备去重，过滤伪文件系统与文件挂载） |
| `disks[].total` | int | 该卷总容量（字节） |
| `disks[].available` | int | 该卷可用空间（字节） |
| `disks[].usage` | float | 该卷使用率（%） |
| `network.rx_bytes` | int | 累计接收字节数 |
| `network.tx_bytes` | int | 累计发送字节数 |
| `network.rx_rate` | int | 接收速率（字节/秒） |
| `network.tx_rate` | int | 发送速率（字节/秒） |
| `uptime` | int | 系统运行时长（秒） |
| `hostname` | string | 主机名 |

**Status Codes:**
- `200 OK` — 正常返回指标
- 采集失败时对应字段返回零值，不返回错误码（前端按零值降级展示）

---

## 7. 热重载机制

### 原理

```
 fsnotify 监听 config.yaml
      │
 收到 Write 事件
      │
  防抖 100ms（聚合连续写入）
      │
 读取新文件 → yaml.Unmarshal
      │
  ┌── 成功？──┐
  │           │
  是           否
  │           │
 加写锁       日志警告
 替换配置      保留旧配置
 释放写锁
 日志记录
```

### 关键设计决策

1. **原子替换**：新配置解析成功后，通过指针原子替换（`atomic.Value` 或 `RWMutex`），确保读取方永远不会看到半份配置
2. **失败不中断**：YAML 解析失败时保留旧配置，仅输出错误日志，服务继续运行
3. **防抖**：100ms 延迟聚合编辑器的连续保存事件，避免频繁重载
4. **无需重启**：所有变更在进程内存中完成，PID 不变，已有连接不受影响

### 前端感知方案

选择 **定时轮询** 而非 WebSocket/SSE，理由：
- 导航配置变更频率极低（天级别）
- 实现简单，不需要维护长连接
- 10 秒轮询对服务器几乎无负担
- 前端检测到配置变化后，不会重新加载页面，仅更新导航树状态

---

## 8. 目录结构

```
dashboard/
├── backend/                  # Go 服务端
│   ├── main.go               # 入口（含 //go:embed）
│   ├── config.go             # 配置结构体 + YAML 解析
│   ├── watcher.go            # fsnotify 文件监听
│   ├── stats.go              # 系统指标采集（/proc 读取 + 内存缓存）
│   ├── handler.go            # HTTP handler（/api/config + /api/stats + 静态文件）
│   └── go.mod                # Go 模块定义
│
├── frontend/                 # Vue 3 前端
│   ├── index.html            # HTML 入口
│   ├── package.json          # 依赖声明
│   ├── vite.config.js        # Vite 构建配置
│   └── src/
│       ├── main.js           # Vue 入口
│       ├── App.vue           # 根组件
│       ├── components/
│       │   ├── StatusBar.vue # 顶部状态条（品牌 + 状态指示 + 时钟）
│       │   ├── Masthead.vue  # 首屏大标题 + 旋转徽章 + 统计
│       │   ├── NavPanel.vue  # 控制条（搜索+网络拨杆）+ 机架
│       │   ├── NavGroup.vue  # 分组（可折叠）
│       │   ├── NavItem.vue   # 站点行（双地址芯片 + 展开详情）
│       │   ├── Rail.vue      # 右侧监视器（真实系统指标）
│       │   ├── Marquee.vue   # 跑马灯
│       │   ├── TrafficCanvas.vue # 背景流量动画
│       │   └── BootScreen.vue # BIOS 开机自检
│       └── style.css         # 单套样式（机架工业风暗色，可读性优化）
│
├── Dockerfile                # 多阶段构建（node → go → alpine）
├── docker-compose.yml        # Docker 部署编排
├── config.yaml               # 导航配置（修改即生效，运行时读取，不嵌入）
├── DESIGN.md                 # ← 本文档
└── README.md                 # 使用说明
```

### 构建产物（build/ 目录，gitignore）

```
dashboard/
└── dashboard                 # 单二进制（Go 编译产物，含嵌入的前端静态文件）
```

- 运行时仅需：`dashboard` 二进制 + `config.yaml`
- 前端 `frontend/dist/` 构建产物已嵌入 `dashboard` 二进制，无需单独分发

### Docker 部署

多阶段构建，最终镜像基于 `alpine`，仅包含二进制 + 挂载的配置文件：

```dockerfile
# 阶段 1：前端构建
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# 阶段 2：Go 编译（含 //go:embed）
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY backend/ ./backend/
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN cd backend && go build -o /dashboard .

# 阶段 3：运行时
FROM alpine:3.20
COPY --from=builder /dashboard /app/dashboard
WORKDIR /app
EXPOSE 8080
ENTRYPOINT ["/app/dashboard"]
```

```yaml
# docker-compose.yml
services:
  dashboard:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml:ro
      - /proc:/host/proc:ro
      - /sys:/host/sys:ro
    environment:
      - HOST_PROC=/host/proc
      - HOST_SYS=/host/sys
    restart: unless-stopped
```

**关键约束：**

- `config.yaml` 必须挂载（`:ro`），它是运行时读取 + fsnotify 热重载的，不能打进镜像
- `/proc`、`/sys` 挂载为只读，供系统指标采集使用
- fsnotify 在 bind mount 上监听的是 inode 变更，宿主机用 `vim` 编辑（先写临时文件再 rename）会换 inode，**watcher 需同时监听 `Create` 事件**，或改为监听目录
- 容器内只能看到挂载进来的 `/proc`，不会暴露宿主机其他信息

---

## 9. 技术决策记录 (ADR)

### ADR-1: 后端选 Go 而非 Node.js/Python

- **背景**：需要静态文件服务 + 配置 API + 文件监听
- **决策**：Go
- **理由**：
  - 编译为单二进制，部署简单
  - 标准库 `net/http` 足以覆盖需求，零框架依赖
  - `fsnotify` 成熟稳定
  - 内存占用低，适合常驻服务
- **后果**：需要 Go 编译环境，构建比解释型语言多一步

### ADR-2: 前端轮询而非长连接

- **背景**：前端需要感知配置变更
- **决策**：10 秒轮询
- **理由**：
  - 导航配置变更频率极低，轮询开销可忽略
  - 实现简单，无需处理 WebSocket 重连/心跳
  - 无额外端口/连接管理
- **后果**：配置变更后最多有 10 秒延迟

### ADR-3: fsnotify 而非定时扫描文件 mtime

- **背景**：需要检测 config.yaml 变更
- **决策**：fsnotify
- **理由**：
  - 操作系统事件驱动，响应及时
  - 不消耗 CPU 周期轮询文件系统
  - 不会漏掉变更（mtime 扫描有间隔窗口）
- **后果**：增加一个外部依赖

### ADR-4: 无 UI 框架，纯 CSS

- **背景**：导航面板不涉及复杂交互
- **决策**：手写 CSS
- **理由**：
  - 组件数量少（3 个），引入 UI 框架（Element Plus / Ant Design）增加构建体积和维护成本
  - 导航面板样式定制化程度高，框架组件反而需要大量覆盖
  - YAGNI
- **后果**：需要手写响应式/主题样式

### ADR-5: 前端静态文件嵌入 Go 二进制

- **背景**：前端构建产物需要与后端一起部署
- **决策**：使用 Go 标准库 `embed` 将 `frontend/dist/` 编译时嵌入二进制
- **理由**：
  - 部署时只需一个二进制文件 + 一个 YAML 配置文件，无需额外分发前端文件
  - Go 1.16+ 标准库能力，零额外依赖
  - 路径不会漂移（二进制路径与运行时解耦）
  - `config.yaml` 不嵌入——保持热重载能力，避免修改 YAML 后需重新编译
- **后果**：
  - 构建流程变为两步：`npm run build` → `go build`
  - 前端变更后需要重新编译 Go 二进制才能生效（仅发布时，开发阶段仍使用 Vite dev server）

### ADR-6: 系统指标从 /proc 直接读取，不依赖外部监控服务

- **背景**：状态条需要展示 CPU、内存、磁盘、网络、运行时长等系统指标
- **决策**：Go 后端直接读取 `/proc`、`/sys`，不接入 Beszel / Prometheus / Portracker 等已有监控服务
- **理由**：
  - 导航面板是轻量工具，不应依赖外部监控服务的可用性和认证
  - `/proc` 读取零外部依赖，Go 标准库 + `syscall.Statfs` 即可覆盖全部指标
  - 减少部署复杂度——不需要配置监控服务地址、API Key、网络连通性
  - 容器内通过挂载宿主机 `/proc`、`/sys` 即可获取真实指标，不受 cgroup 隔离限制
- **后果**：
  - 指标精度和丰富度不如专业监控（无历史曲线、无告警）
  - 需要 Docker 部署时额外挂载 `/proc`、`/sys`
  - 若后续需要更丰富的监控，应引导用户使用已有的 Beszel/Grafana，而非在本项目中扩展

### ADR-7: Docker 部署采用多阶段构建 + 卷挂载

- **背景**：项目需要支持 Docker 部署，同时保持单二进制 + 配置文件的简洁部署模型
- **决策**：多阶段构建（node → golang → alpine），运行时挂载 `config.yaml`、`/proc`、`/sys`
- **理由**：
  - 多阶段构建将前端编译、Go 编译、运行时分离，最终镜像仅含一个二进制（~15MB）
  - `config.yaml` 必须挂载而非打入镜像——保持热重载能力，修改配置无需重建镜像
  - `/proc`、`/sys` 只读挂载，供系统指标采集，不引入特权模式
  - 通过 `HOST_PROC` / `HOST_SYS` 环境变量适配路径，裸机部署时无需设置
- **后果**：
  - fsnotify 在 bind mount 上可能因 inode 变更丢失事件，watcher 需同时监听 `Create` 事件
  - 构建需要 Docker 环境（或本地分别执行 `npm run build` + `go build`）

---

## 10. 交付清单

| 阶段 | 产物 | 说明 |
|------|------|------|
| 后端 | `backend/main.go` | 入口（含 `//go:embed`），启动服务 |
| 后端 | `backend/config.go` | 配置加载 |
| 后端 | `backend/watcher.go` | 热重载 |
| 后端 | `backend/stats.go` | 系统指标采集（/proc 读取 + 内存缓存） |
| 后端 | `backend/handler.go` | API（/api/config + /api/stats）+ 静态文件服务 |
| 后端 | `backend/go.mod` | 模块依赖 |
| 前端 | `frontend/src/main.js` | Vue 入口 |
| 前端 | `frontend/src/App.vue` | 根组件（数据编排 + 布局 + 快捷键） |
| 前端 | `frontend/src/api.js` | /api/config + /api/stats 请求 |
| 前端 | `frontend/src/utils.js` | 地址解析 + 格式化工具 |
| 前端 | `frontend/src/components/StatusBar.vue` | 顶部状态条（品牌 + 状态指示 + 时钟） |
| 前端 | `frontend/src/components/Masthead.vue` | 首屏大标题 + 旋转徽章 + 统计 |
| 前端 | `frontend/src/components/NavPanel.vue` | 控制条（搜索 + 网络拨杆）+ 机架 |
| 前端 | `frontend/src/components/NavGroup.vue` | 分组（可折叠） |
| 前端 | `frontend/src/components/NavItem.vue` | 站点行（双地址芯片 + 展开详情） |
| 前端 | `frontend/src/components/Rail.vue` | 右侧监视器（真实系统指标） |
| 前端 | `frontend/src/components/Marquee.vue` | 跑马灯 |
| 前端 | `frontend/src/components/TrafficCanvas.vue` | 背景流量动画 |
| 前端 | `frontend/src/components/BootScreen.vue` | BIOS 开机自检 |
| 前端 | `frontend/src/style.css` | 单套样式（机架工业风暗色，可读性优化） |
| 前端 | `frontend/index.html` | HTML 入口 |
| 前端 | `frontend/vite.config.js` | 构建配置 |
| 前端 | `frontend/package.json` | 依赖 |
| 部署 | `Dockerfile` | 多阶段构建（node → go → alpine） |
| 部署 | `docker-compose.yml` | Docker 部署编排（挂载 config.yaml + /proc + /sys） |
| 配置 | `config.yaml` | 示例导航配置 |
