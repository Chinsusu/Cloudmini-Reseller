FROM golang:1.22-alpine AS builder
WORKDIR /app

# Copy go.work and all local modules
COPY go.work go.work.sum* ./
COPY pkg/ pkg/
COPY services/user-service/ services/user-service/

WORKDIR /app/services/user-service
RUN go build -ldflags="-s -w" -o /bin/user-service ./cmd/server/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/user-service /bin/user-service
EXPOSE 8081
USER nobody
ENTRYPOINT ["/bin/user-service"]
