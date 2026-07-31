# 阶段 1：前端构建
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm install --registry=https://registry.npmmirror.com
COPY frontend/ ./
RUN npm run build

# 阶段 2：Go 编译（//go:embed 将 dist 打包进二进制）
FROM golang:1.25-alpine AS builder
WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct
COPY backend/ ./backend/
COPY --from=frontend /app/frontend/dist ./backend/frontend/dist
RUN cd backend && go build -o /dashboard .

# 阶段 3：运行时（仅一个二进制 + 示例配置兜底）
FROM alpine:3.20
COPY --from=builder /dashboard /app/dashboard
# 内置示例配置，镜像可独立启动；compose 挂载 ./config.yaml 后自动覆盖
COPY config.yaml /app/config.yaml
WORKDIR /app
EXPOSE 8080
ENTRYPOINT ["/app/dashboard"]
