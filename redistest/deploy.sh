# /bin/bash
CGO_ENABLED=0 go build -o redistest -ldflags="-s -w" main.go
docker build . -t njucz/redistest:latest
docker push docker.io/njucz/redistest:latest
# docker run --name redistest -p 8080:8080 njucz/redistest:latest