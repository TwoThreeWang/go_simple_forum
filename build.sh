#!/bin/bash
cd "$(dirname "$0")"
echo '开始拉取最新代码'
git pull origin main
#echo '删除镜像'
#docker images --filter=reference="zhulink" -a -q | xargs docker rmi -f
echo '打包镜像'
docker build -t zhulink:latest .
docker-compose down
echo '启动容器'
docker-compose up -d --remove-orphans
#echo '停止并删除旧容器'
#docker rm -f zhulink
#echo '启动容器'
#docker run --name zhulink -d -v ./.env:/.env -p 32919:32919 -v ./templates:/templates -v ./static:/static:rw --log-opt max-size=50m --restart=always zhulink:latest
echo '清理不再使用的镜像、容器和数据卷'
docker system prune --all --force --volumes --filter "label!=keep=true"