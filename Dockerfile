FROM golang:1.14-alpine as backend

# Install dependencies
WORKDIR /go/src/github.com/abatilo/github-action-locks
COPY ./go.mod ./go.sum ./
RUN go mod download

# Build artifacts
WORKDIR /go/src/github.com/abatilo/github-action-locks
COPY ./main.go ./main.go
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/github-action-locks main.go

FROM scratch
# SSL Certs
COPY --from=backend /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy our static executable
COPY --from=backend /go/bin/github-action-locks /go/bin/github-action-locks

# Run the hello binary.
ENTRYPOINT ["/go/bin/github-action-locks"]
