FROM golang:1.19-alpine3.17 as builder

WORKDIR /app

COPY . .

# used for build sqlite
RUN apk add --update gcc musl-dev

RUN CGO_ENABLED=1 GOOS=linux go build -a -v -installsuffix cgo --tags "sqlite_foreign_keys" ./cmd/scanner
RUN CGO_ENABLED=1 GOOS=linux go build -a -v -installsuffix cgo --tags "sqlite_foreign_keys" ./cmd/api
RUN CGO_ENABLED=1 GOOS=linux go build -a -v -installsuffix cgo --tags "sqlite_foreign_keys" ./cmd/proxy

FROM alpine:3.17
RUN apk add ca-certificates
#FIX of alpine can't find binary file
RUN apk add --no-cache libc6-compat
RUN apk add nmap
COPY --from=builder /app/scanner /usr/bin/
COPY --from=builder /app/api /usr/bin/
COPY --from=builder /app/proxy /usr/bin/

COPY --from=builder /app/db /db
COPY --from=builder /app/creds /creds