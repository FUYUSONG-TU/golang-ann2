# ---------- 阶段 1：构建前端 ----------
FROM node:20-alpine AS web
WORKDIR /web
COPY web/package*.json web/.npmrc ./
RUN npm install --no-fund --no-audit
COPY web/ ./
RUN npm run build

# ---------- 阶段 2：构建后端（前端产物嵌入二进制） ----------
FROM golang:1.21-alpine AS builder
WORKDIR /app
ENV CGO_ENABLED=0 \
    GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=off
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /web/dist ./web/dist
RUN go build -ldflags="-s -w" -o main .

# ---------- 阶段 3：运行时 ----------
FROM alpine:3.19
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder /app/main /app/main

EXPOSE 80
CMD ["/app/main"]
