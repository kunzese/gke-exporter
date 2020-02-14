FROM golang:1.13.8-alpine as builder
RUN apk add --no-cache git
WORKDIR /go/src/github.com/kunzese/gke-exporter
COPY . .
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -a -o /app cmd/main.go

FROM gcr.io/distroless/base
COPY --from=builder /app /app
CMD ["/app"]
