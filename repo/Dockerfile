# Build the manager binary
FROM golang:1.24.1 AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace

# Copy the Go Modules manifests
COPY repo/go.mod go.mod
COPY repo/go.sum go.sum
# Copy the `common` directory to the Docker image
COPY common ./common

# Ensure the `replace` directive in go.mod points to the correct path
RUN sed -i 's|replace codek7/common => ../common|replace codek7/common => ./common|' go.mod

# Download Go modules
RUN go mod download

# Copy the source code into the Docker image
COPY repo/internal ./internal
COPY repo/pkg ./pkg
COPY repo/cmd ./cmd
COPY repo/db ./db
# Build the Go binary
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -a -o repo cmd/main.go

# Use a minimal base image for the final image

CMD ["./repo"]


