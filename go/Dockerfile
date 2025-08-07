FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -o daytona-proxy .

FROM alpine:3.22

WORKDIR /app
COPY --from=builder /app/daytona-proxy .
EXPOSE 3000
CMD ["./daytona-proxy"]
