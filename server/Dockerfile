FROM mirror.houlang.cloud/dh/golang:1.25.8-alpine AS builder

WORKDIR /src

RUN apk add --no-cache build-base

ARG GOPROXY=https://goproxy.cn,direct
ENV GOPROXY=$GOPROXY

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -o /out/hl6-server ./cmd/server

FROM mirror.houlang.cloud/dh/alpine:3.22

RUN apk add --no-cache ca-certificates tzdata libgcc

WORKDIR /app
COPY --from=builder /out/hl6-server /app/server

EXPOSE 8080
ENTRYPOINT ["/app/server"]
