FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY services/storage/go.mod services/storage/go.sum ./
COPY pkg/ /pkg/
RUN go mod download
COPY services/storage/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /server /server
EXPOSE 8088
ENTRYPOINT ["/server"]
