FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
ENV GOPROXY=https://package-mirror.liara.ir/repository/go/
ENV GOSUMDB=off
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

FROM alpine:latest
RUN sed -i 's/dl-cdn.alpinelinux.org/mirror.arvancloud.ir/g' /etc/apk/repositories && apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
