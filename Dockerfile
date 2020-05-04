FROM golang:latest

RUN apt-get update

RUN apt-get install -y netcat

WORKDIR /app

COPY go.mod /app

RUN go mod download

COPY . /app
COPY .env /app

RUN go get github.com/githubnemo/CompileDaemon

ENTRYPOINT ["/app/docker/utils/entrypoint.sh"]
