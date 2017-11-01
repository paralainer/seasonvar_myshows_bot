FROM golang:1.8.1

WORKDIR /go/src/seasonvar_myshows_bot
COPY . .

RUN go-wrapper download   # "go get -d -v ./..."
RUN go-wrapper install    # "go install -v ./..."

CMD ["go-wrapper", "run"]