FROM --platform=$BUILDPLATFORM golang:1.25 as builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETBIN
ARG LDFLAGS

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/

RUN <<EOF
set -ex
CGO_ENABLED=0 \
GOOS=$TARGETOS \
GOARCH=$TARGETARCH \
go build \
    -trimpath \
    -ldflags="$LDFLAGS" \
    -a \
    -o /workspace/app \
    ./cmd/$TARGETBIN
EOF


FROM gcr.io/distroless/static:nonroot

WORKDIR /
COPY --from=builder /workspace/app /app

USER 65532:65532
ENTRYPOINT ["/app"]
