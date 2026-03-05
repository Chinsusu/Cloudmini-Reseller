FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum* ./
COPY pkg/ pkg/
COPY services/reseller-service/ services/reseller-service/
WORKDIR /app/services/reseller-service
RUN go build -ldflags="-s -w" -o /bin/reseller-service ./cmd/server/
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/reseller-service /bin/app
EXPOSE 8084
USER nobody
ENTRYPOINT ["/bin/app"]
