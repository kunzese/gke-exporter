FROM golang:1.19.5-alpine as builder
ARG VERSION
RUN apk add --no-cache git
WORKDIR /go/src/github.com/kunzese/gke-exporter
COPY . .
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build \
      -trimpath \
      -ldflags "-s -w -X main.version=$VERSION" \
      -o /app \
      cmd/main.go

FROM gcr.io/distroless/base
COPY --from=builder /app /app
CMD ["/app"]
