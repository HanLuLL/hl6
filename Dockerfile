FROM mirror.houlang.cloud/dh/node:22-alpine AS web-builder

WORKDIR /src/web

COPY web/package.json web/package-lock.json ./
RUN npm ci

COPY web/ ./
RUN npm run build

FROM mirror.houlang.cloud/dh/golang:1.25.6-alpine AS server-builder

WORKDIR /src

RUN apk add --no-cache build-base

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -o /out/hl6-server ./cmd/server

FROM mirror.houlang.cloud/dh/alpine:3.22

RUN apk add --no-cache ca-certificates tzdata libgcc

WORKDIR /app

COPY --from=server-builder /out/hl6-server /app/server
COPY --from=web-builder /src/web/dist /app/web/dist

EXPOSE 8080

ENTRYPOINT ["/app/server"]
