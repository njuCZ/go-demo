# /bin/bash
CGO_ENABLED=0 go build -o msi -ldflags="-s -w" main.go
docker build . -t njucz/msi:latest
docker push docker.io/njucz/msi:latest
# docker run --name msi -p 8080:8080 njucz/msi:latest