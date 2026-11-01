#!/bin/bash
# tikuAdapter 云端独立部署脚本
# 用法：在云服务器上执行 bash deploy.sh
# 前置：tiku-adapter-image.tar.gz 与 config.yaml 已上传到同目录

set -e
cd "$(dirname "$0")"

echo "=== tikuAdapter 云端部署 ==="

# 1. 检查 Docker
if ! command -v docker &> /dev/null; then
    echo "❌ Docker 未安装，请先安装 Docker"
    exit 1
fi
echo "✓ Docker 已安装"

# 2. 加载镜像
if [ -f "tiku-adapter-image.tar.gz" ]; then
    echo "加载镜像（可能需要 1-2 分钟）..."
    docker load < tiku-adapter-image.tar.gz
    echo "✓ 镜像加载完成"
else
    echo "⚠️  未找到 tiku-adapter-image.tar.gz，跳过加载（假设镜像已存在）"
fi

# 3. 准备数据目录
mkdir -p data
echo "✓ 数据目录已创建"

# 4. 停止旧容器（如有）
if docker ps -a --format '{{.Names}}' | grep -q '^tiku-adapter$'; then
    echo "停止并删除旧容器..."
    docker stop tiku-adapter 2>/dev/null || true
    docker rm tiku-adapter 2>/dev/null || true
fi

# 5. 启动新容器
echo "启动容器..."
docker run -d \
  --name tiku-adapter \
  --restart unless-stopped \
  -p 8060:8060 \
  -v "$(pwd)/data:/app/data" \
  -v "$(pwd)/config.yaml:/app/config.yaml" \
  tiku-adapter:latest

# 6. 等待启动并验证
echo "等待服务启动..."
sleep 5
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8060/ || echo "000")
if [ "$HTTP_CODE" = "200" ]; then
    echo "✓ 服务已启动: http://localhost:8060 (HTTP $HTTP_CODE)"
else
    echo "⚠️  服务返回 HTTP $HTTP_CODE，请检查日志: docker logs tiku-adapter"
fi

echo ""
echo "=== 部署完成 ==="
echo "搜题 API: http://<服务器IP>:8060/adapter-service/search"
echo "管理面板: http://<服务器IP>:8060 (admin / 密码见 TIKU_ADMIN_PASSWORD 环境变量)"
echo "查看日志: docker logs -f tiku-adapter"
