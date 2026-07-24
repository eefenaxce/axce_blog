#!/bin/bash
set -e

# 服务器部署脚本
# 用法：
#   1. 修改下面的 IMAGE、CONTAINER_NAME、PORT
#   2. 把 config.yaml 和 docker-compose.yml 放在同一目录
#   3. 运行：chmod +x deploy.sh && ./deploy.sh

IMAGE="your-dockerhub-username/axce-blog:latest"
CONTAINER_NAME="axce-blog"
PORT="8080"

echo "Pulling latest image: $IMAGE"
docker pull "$IMAGE"

echo "Stopping existing container: $CONTAINER_NAME"
docker stop "$CONTAINER_NAME" 2>/dev/null || true
docker rm "$CONTAINER_NAME" 2>/dev/null || true

echo "Starting new container: $CONTAINER_NAME"
docker run -d \
  --name "$CONTAINER_NAME" \
  --restart unless-stopped \
  -p "${PORT}:8080" \
  -v "$(pwd)/config.yaml:/app/config.yaml:ro" \
  -v "$(pwd)/themes:/app/themes" \
  -v "$(pwd)/static:/app/web/build/client/static" \
  "$IMAGE"

echo "Container started. Logs:"
docker logs -f "$CONTAINER_NAME"