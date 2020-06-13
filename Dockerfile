FROM golang:1.14-alpine as backend

# Install dependencies
WORKDIR /go/src/github.com/abatilo/github-action-locks
COPY ./go.mod ./go.sum ./
COPY ./vendor ./vendor

# Build artifacts
WORKDIR /go/src/github.com/abatilo/github-action-locks
COPY ./main.go ./main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=vendor -ldflags="-w -s" -o /go/bin/github-action-locks main.go

FROM alpine:3
# SSL Certs
COPY --from=backend /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy our static executable
COPY --from=backend /go/bin/github-action-locks /go/bin/github-action-locks
COPY lock.sh unlock.sh action.yml /
