# --- Build stage ---
FROM golang:1.25.1 AS builder
WORKDIR /app

COPY go.mod go.sum ./
# (ออปชัน แต่ช่วยได้) ให้ go ดึง toolchain อัตโนมัติถ้าจำเป็น
ENV GOTOOLCHAIN=auto
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o server .

# --- Runtime stage ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata curl && adduser -D -H appuser
ENV TZ=Asia/Bangkok
WORKDIR /app
COPY --from=builder /app/server .
USER appuser
EXPOSE 8080
ENTRYPOINT ["./server"]
