FROM golang:1.26.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum* ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/store-api ./cmd/store-api

FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/store-api /app/store-api

ENV APP_ENV=production
ENV PORT=10000

EXPOSE 10000

CMD ["/app/store-api"]
