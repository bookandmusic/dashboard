# 阶段 1：前端构建（固定到构建平台，产物与 CPU 架构无关，跨平台复用）
FROM --platform=${BUILDPLATFORM} node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install --registry=https://registry.npmmirror.com
COPY frontend/ ./
RUN npm run build

# 阶段 2：Go 编译（//go:embed 将 dist 打包进二进制）
# 固定到构建平台：go 工具链原生运行（amd64 runner），通过 GOARCH 交叉编译，无需 QEMU
FROM --platform=${BUILDPLATFORM} golang:1.25-alpine AS builder
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct
ARG TARGETARCH
COPY backend/ ./backend/
COPY --from=frontend /app/frontend/dist ./backend/frontend/dist
RUN cd backend && CGO_ENABLED=0 GOARCH=${TARGETARCH} go build -trimpath -ldflags="-s -w" -o /dashboard .

# 阶段 3：运行时（仅一个二进制 + 示例配置兜底）
FROM alpine:3.20
COPY --from=builder /dashboard /app/dashboard
# 内置示例配置，镜像可独立启动；compose 挂载 ./config.yaml 后自动覆盖
COPY config.yaml /app/config.yaml
WORKDIR /app
EXPOSE 8080
ENTRYPOINT ["/app/dashboard"]
