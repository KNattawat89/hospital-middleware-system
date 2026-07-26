FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go tool swag init --parseDependency --parseInternal
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/server .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /app/bin/server ./server

EXPOSE 8080

ENTRYPOINT ["./server"]
