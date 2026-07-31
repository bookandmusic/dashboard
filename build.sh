#!/usr/bin/env bash
# 构建脚本：前端构建 → 暂存 dist 供 go:embed → 编译单二进制
set -euo pipefail
cd "$(dirname "$0")"

echo "==> [1/3] 前端构建"
(cd frontend && npm run build)

echo "==> [2/3] 暂存 dist 到 backend/frontend/dist（供 //go:embed）"
mkdir -p backend/frontend
rm -rf backend/frontend/dist
cp -r frontend/dist backend/frontend/dist

echo "==> [3/3] Go 编译"
(cd backend && go build -o ../dashboard .)

echo "==> 完成：./dashboard（单二进制，已嵌入前端）"
echo "    运行：./dashboard  （默认读取 ./config.yaml，监听 :8080）"
