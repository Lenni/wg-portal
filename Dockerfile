# Dockerfile References: https://docs.docker.com/engine/reference/builder/
# This dockerfile uses a multi-stage build system to reduce the image footprint.

######-
# Start from the latest golang base image as builder image (only used to compile the code)
######-
FROM golang:1.16 as builder

ARG BUILD_IDENTIFIER
ENV ENV_BUILD_IDENTIFIER=$BUILD_IDENTIFIER

ARG BUILD_VERSION
ENV ENV_BUILD_VERSION=$BUILD_VERSION

# populated by BuildKit
ARG TARGETPLATFORM
ENV ENV_TARGETPLATFORM=$TARGETPLATFORM

RUN mkdir /build

# Copy the source from the current directory to the Working Directory inside the container
ADD . /build/

# Set the Current Working Directory inside the container
WORKDIR /build

# Workaround for failing travis-ci builds
RUN rm -rf ~/go; rm -rf go.sum

# Download dependencies
RUN curl -L https://git.prolicht.digital/pub/healthcheck/-/releases/v1.0.1/downloads/binaries/hc -o /build/hc; \
    chmod +rx /build/hc; \
    echo "Building version: $ENV_BUILD_IDENTIFIER-$ENV_BUILD_VERSION"

## Add the wait script to the image
ADD https://github.com/ufoscout/docker-compose-wait/releases/download/2.9.0/wait wait
RUN chmod +x wait

# Build the Go app
RUN echo "Building version '$ENV_BUILD_IDENTIFIER-$ENV_BUILD_VERSION' for platform $ENV_TARGETPLATFORM"; make build

######-
# Here starts the main image
######-
FROM alpine:3

# Setup timezone
ENV TZ=Europe/Vienna

# Import linux stuff from builder.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Import healthcheck binary
COPY --from=builder /build/hc /app/hc
COPY --from=builder /build/wait /app/wait

# Copy binaries
COPY --from=builder /build/dist/wg-portal /app/wg-portal

# Set the Current Working Directory inside the container
WORKDIR /app

# Command to run the executable
CMD /app/wait && /app/wgportal

HEALTHCHECK --interval=30s --timeout=5s --start-period=30s --retries=3 CMD [ "/app/hc", "http://localhost:11223/health" ]
