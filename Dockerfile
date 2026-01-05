FROM --platform=$BUILDPLATFORM golang:1.25 as builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETBIN
ARG GCFLAGS
ARG LDFLAGS
ARG BUILD_TYPE=prod

WORKDIR /go

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/ cmd/
COPY internal/ internal/
COPY pkg/ pkg/

# Install dlv only for debug
RUN <<EOF
if [ "$BUILD_TYPE" = "debug" ]
then
    CGO_ENABLED=0 \
    GOOS=$TARGETOS \
    GOARCH=$TARGETARCH \
    go install github.com/go-delve/delve/cmd/dlv@latest
    # Work-around for Go crossplatform installations.
    if [ -d "bin/${TARGETOS}_${TARGETARCH}" ]
    then
        mv "bin/${TARGETOS}_${TARGETARCH}"/* bin/
    fi
fi
EOF

RUN <<EOF
set -ex
CGO_ENABLED=0 \
GOOS=$TARGETOS \
GOARCH=$TARGETARCH \
go build \
    -trimpath \
    -gcflags="$GCFLAGS" \
    -ldflags="$LDFLAGS" \
    -a \
    -o bin/app \
    ./cmd/$TARGETBIN
EOF


FROM gcr.io/distroless/static:nonroot AS prod

WORKDIR /
COPY --from=builder /go/bin/app /app

ENTRYPOINT ["/app"]
CMD ["serve"]


FROM gcr.io/distroless/static:debug-nonroot AS debug

WORKDIR /
COPY --from=builder /go/bin/app /app
COPY --from=builder /go/bin/dlv /usr/local/bin/dlv

EXPOSE 2345

ENTRYPOINT ["/usr/local/bin/dlv"]
CMD ["exec", "/app", "--headless", "--listen=:2345", "--api-version=2", "--accept-multiclient", "--continue", "--log", "--", "serve"]
