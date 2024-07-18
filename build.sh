git pull origin main
docker build -t zhulink:latest .
docker run --name zhulink -d --env-file .env -p 32912:32912 --log-opt max-size=50m --restart=always zhulink:latest