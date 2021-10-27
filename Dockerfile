FROM golang:1.17.2-alpine AS builder

WORKDIR $GOPATH/src/github.com/Trojan295/organizer-bot
ADD . .
RUN apk add --no-cache build-base && \
    go build -ldflags="-linkmode external -extldflags -static" -o /go/bin/organizer-bot cmd/organizer-bot/main.go


FROM alpine:3.14

RUN apk add --no-cache tzdata ca-certificates
COPY --from=builder /go/bin/organizer-bot /go/bin/organizer-bot

ENTRYPOINT ["/go/bin/organizer-bot"]
