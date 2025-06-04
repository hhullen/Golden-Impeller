FROM golang:1.24.3-alpine3.22 AS builder

WORKDIR /build
COPY . .

RUN go build ./cmd/migrator

FROM alpine:3.22.0

WORKDIR /app
COPY --from=builder /build .

ENV RUNNING_IN_CONTAINER=true

ENTRYPOINT [ "./migrator" ]

