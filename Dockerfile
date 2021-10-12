FROM golang:1.17.2-bullseye AS builder

WORKDIR $GOPATH/src/github.com/Trojan295/organizer-bot
ADD . .

RUN apt-get update && \
    apt-get install -y ca-certificates && \
    go build -ldflags="-linkmode external -extldflags -static" -o /go/bin/organizer-bot cmd/organizer-bot/main.go


FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /go/bin/organizer-bot /go/bin/organizer-bot

ENTRYPOINT ["/go/bin/organizer-bot"]
