FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY go.work go.work.sum* ./
COPY pkg/ pkg/
COPY infrastructure/ infrastructure/
COPY services/billing-service/ services/billing-service/
WORKDIR /app/services/billing-service
RUN go build -ldflags="-s -w" -o /bin/billing-service ./cmd/server/
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /bin/billing-service /bin/app
EXPOSE 8085
USER nobody
ENTRYPOINT ["/bin/app"]
