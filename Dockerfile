FROM golang:1.12.1

ENV GO111MODULE=on

WORKDIR /seasonvar_myshows_bot

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main

ENTRYPOINT ["/seasonvar_myshows_bot/main"]