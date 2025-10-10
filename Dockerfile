FROM golang:1.25.1-alpine AS builder

RUN apk add --no-cache make git ca-certificates

WORKDIR /build

COPY . .

RUN make install-tools
RUN make generate
RUN make build

FROM alpine:latest

RUN apk --no-cache add ca-certificates && \
    addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

WORKDIR /app

COPY --from=builder /build/bin/server .

RUN chown -R appuser:appuser /app

USER appuser

EXPOSE 8080

ENTRYPOINT ["/app/server"]

