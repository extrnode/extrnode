FROM golang:1.18 as builder

WORKDIR /app

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -v -installsuffix cgo ./cmd/indexer
RUN CGO_ENABLED=0 GOOS=linux go build -a -v -installsuffix cgo ./cmd/api

FROM alpine:3.16
RUN apk add ca-certificates
#FIX of alpine can't find binary file
RUN apk add --no-cache libc6-compat
COPY --from=builder /app/indexer /usr/bin/
COPY --from=builder /app/api /usr/bin/