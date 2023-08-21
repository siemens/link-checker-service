# build
FROM golang:1.21 AS builder
WORKDIR /sample-ui/
# cache dependencies (no go.sum in this project)
COPY go.mod ./
RUN go mod download
# now build the whole thing
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -v -o sample-ui

# user
FROM alpine:latest as alpine
# create a user
RUN addgroup -S lcsgroup && adduser -S lcsuser -G lcsgroup

# run
FROM scratch
EXPOSE 8092

# copy the user
COPY --from=alpine /etc/passwd /etc/passwd
USER lcsuser

COPY --from=builder /sample-ui/sample-ui .
ENTRYPOINT ["/sample-ui"]
