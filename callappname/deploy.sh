# /bin/bash
CGO_ENABLED=0 go build -o callappname -ldflags="-s -w" main.go
docker build . -t njucz/callappname:latest
docker push docker.io/njucz/callappname:latest
# docker run --name callappname -p 8080:8080 njucz/callappname:latest