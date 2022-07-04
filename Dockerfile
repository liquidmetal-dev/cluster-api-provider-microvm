# Build the manager binary
FROM golang:1.17 as builder

ARG goproxy=https://proxy.golang.org
ENV GOPROXY=$goproxy

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY ./ ./

# Build
ARG package=.
ARG ARCH
ARG LDFLAGS
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -ldflags "${LDFLAGS} -extldflags '-static'"  -o manager ${package}

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:latest
WORKDIR /
COPY --from=builder /workspace/manager .
USER nobody

ENTRYPOINT ["/manager"]
