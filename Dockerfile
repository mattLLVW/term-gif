FROM golang:latest

WORKDIR /app

COPY . /app
COPY .env /app

RUN go mod download

RUN go get github.com/githubnemo/CompileDaemon

ENTRYPOINT CompileDaemon -directory=. -build="go build main.go" -command=./main
