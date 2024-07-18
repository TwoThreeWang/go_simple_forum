echo '开始拉取最新代码'
git pull origin main
echo '打包镜像'
docker build -t zhulink:latest .
echo '停止并删除旧容器'
docker rm -f zhulink
echo '启动容器'
docker run --name zhulink -d -v ./.env:/.env -p 32919:32919 --log-opt max-size=50m --restart=always zhulink:latest