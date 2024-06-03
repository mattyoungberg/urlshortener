FROM golang:1.19 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -o urlshortener .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/urlshortener /app/urlshortener

EXPOSE 8080

HEALTHCHECK --interval=10s --retries=3 \
CMD wget --quiet --tries=1 --timeout=2 --spider http://localhost:8080/api/v1/health || exit 1

CMD ["/app/urlshortener"]