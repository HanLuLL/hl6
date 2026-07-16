FROM golang:1.25.8-alpine AS builder

WORKDIR /src

RUN apk add --no-cache build-base

ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=$GOPROXY

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
RUN go test ./internal/auth ./internal/middleware ./internal/migration ./internal/repository ./internal/service ./internal/handler
RUN CGO_ENABLED=1 GOOS=linux go build -o /out/hl6-server ./cmd/server
RUN CGO_ENABLED=1 GOOS=linux go build -o /out/hl6-admin ./cmd/hl6-admin

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata libgcc postgresql16-client

WORKDIR /app
COPY --from=builder /out/hl6-server /app/server
COPY --from=builder /out/hl6-admin /app/hl6-admin

EXPOSE 8080

ENTRYPOINT ["/app/server"]
