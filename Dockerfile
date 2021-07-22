# build
FROM golang:1.16 AS builder
WORKDIR /link-checker-service/
# cache dependencies
COPY go.mod go.sum ./
RUN go mod download
# now build the whole thing
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go test -v ./...
RUN CGO_ENABLED=0 GOOS=linux go build -v -o link-checker-service
RUN ./link-checker-service version

# TLS certificates & user
FROM alpine:latest as alpine
RUN apk --no-cache add ca-certificates
# create a user
RUN addgroup -S lcsgroup && adduser -S lcsuser -G lcsgroup

# run
FROM scratch
EXPOSE 8080

# copy tls certificates
COPY --from=alpine /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# copy the user
COPY --from=alpine /etc/passwd /etc/passwd
USER lcsuser

COPY --from=builder /link-checker-service/link-checker-service .
COPY .link-checker-service.toml .
ENTRYPOINT ["/link-checker-service", "serve", "--config", "./.link-checker-service.toml"]
