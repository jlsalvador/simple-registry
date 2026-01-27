# Uses precompiled binaries from build/ directory
#
# Usage:
#   make build  # Compile binaries first
#   docker build --platform=linux/amd64 -t simple-registry:latest .

FROM gcr.io/distroless/static:nonroot

ARG TARGETOS
ARG TARGETARCH

WORKDIR /

# Copy precompiled binary from build directory
# Pattern matches: build/simple-registry_X.Y.Z_linux-amd64
COPY build/simple-registry_*_${TARGETOS}-${TARGETARCH} /usr/bin/simple-registry

ENTRYPOINT ["/usr/bin/simple-registry"]
CMD ["serve"]
