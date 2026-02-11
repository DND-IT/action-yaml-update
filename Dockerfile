FROM golang:1.23-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /yaml-update ./cmd/yaml-update

FROM alpine:3.20

RUN apk add --no-cache git

COPY --from=builder /yaml-update /yaml-update

ENTRYPOINT ["/yaml-update"]
