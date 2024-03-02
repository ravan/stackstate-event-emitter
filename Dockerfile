FROM golang:1.21-alpine as builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -trimpath -o /app/eventemitter ./cmd/eventemitter/main.go

FROM alpine:3.19.1

USER 1001

WORKDIR /app

COPY --from=builder /app/eventemitter .

CMD ["/app/eventemitter"]
