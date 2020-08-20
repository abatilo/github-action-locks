FROM alpine:3
# Copy our static executable
COPY dist/github-action-locks /go/bin/github-action-locks
COPY lock.sh unlock.sh action.yml /
