FROM golang:1.22-alpine AS builder
WORKDIR /build
COPY services/support/go.mod services/support/go.sum ./
COPY pkg/ /pkg/
RUN go mod download
COPY services/support/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /server ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /server /server
EXPOSE 8092
ENTRYPOINT ["/server"]
