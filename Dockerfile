# ビルドステージ
FROM golang:1.24.1-alpine AS builder

WORKDIR /app

# 依存関係の事前ダウンロード（キャッシュ効率化）
COPY go.mod go.sum ./
RUN go mod download

# ソースコードのビルド
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o mysql-schema-explorer-mcp .

# 実行ステージ
FROM alpine:latest

# CA証明書の追加（HTTPS通信用）
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# ビルド済みバイナリのコピー
COPY --from=builder /app/mysql-schema-explorer-mcp .

# エントリーポイントの設定
ENTRYPOINT ["./mysql-schema-explorer-mcp"]
