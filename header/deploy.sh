# /bin/bash
CGO_ENABLED=0 go build -o print-header -ldflags="-s -w" main.go
docker build . -t njucz/print-header:latest
docker push docker.io/njucz/print-header:latest
# docker run --name print-header -p 8080:8080 njucz/print-header:latest