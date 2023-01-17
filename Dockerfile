FROM golang:1.19 as builder

WORKDIR /app

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -v -installsuffix cgo ./cmd/scanner
RUN CGO_ENABLED=0 GOOS=linux go build -a -v -installsuffix cgo ./cmd/api

FROM alpine:3.17
RUN apk add ca-certificates
#FIX of alpine can't find binary file
RUN apk add --no-cache libc6-compat
RUN apk add nmap
COPY --from=builder /app/scanner /usr/bin/
COPY --from=builder /app/api /usr/bin/

COPY --from=builder /app/db /db
COPY --from=builder /app/certs /certs