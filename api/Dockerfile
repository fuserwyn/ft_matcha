FROM golang:1.23-alpine AS builder

WORKDIR /app
ENV GOTOOLCHAIN=auto
ENV GOMAXPROCS=1

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOFLAGS="-p=1" go build -trimpath -ldflags="-s -w" -o /api ./cmd/api

FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=builder /api .

EXPOSE 8080
CMD ["./api"]
