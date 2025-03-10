FROM golang:1.24 AS build
WORKDIR /go/src
COPY go ./go
COPY main.go .
COPY go.sum .
COPY go.mod .

ENV CGO_ENABLED=0

RUN go mod tidy
RUN go build -o datamonkey .

FROM alpine:3.19 AS runtime
ENV GIN_MODE=release

# Install shadow package for su command
RUN apk add --no-cache shadow

# Create slurm user and group with the same UID/GID as in the Slurm container
RUN addgroup -g 990 slurm && \
    adduser -D -u 990 -G slurm slurm

# Create directories for JWT keys with proper permissions
RUN mkdir -p /jwt_keys /var/spool/slurm/statesave && \
    chown slurm:slurm /jwt_keys /var/spool/slurm/statesave && \
    chmod 755 /jwt_keys /var/spool/slurm/statesave

# TODO: this is specific to a scheduler type, make it dependent on the env var?
# Create a directory for a local copy of the JWT key if needed
RUN mkdir -p /usr/local/etc/jwt && \
    chown slurm:slurm /usr/local/etc/jwt && \
    chmod 700 /usr/local/etc/jwt

# Copy the binary from the build stage
COPY --from=build /go/src/datamonkey /usr/local/bin/

# Add after copying the binary
COPY docker-entrypoint.sh /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

EXPOSE 9300/tcp

# Change the entrypoint
ENTRYPOINT ["/usr/local/bin/docker-entrypoint.sh", "/usr/local/bin/datamonkey"]
