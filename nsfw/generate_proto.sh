#!/bin/bash

# Generate Python gRPC files from proto definition
# Make sure grpcio-tools is installed: pip install grpcio-tools

echo "Generating Python gRPC files from repo.proto..."

python -m grpc_tools.protoc \
    --python_out=. \
    --grpc_python_out=. \
    --proto_path=. \
    ../repo/pkg/pb/repo.proto

echo "Generated files:"
echo "- repo_pb2.py (message definitions)"
echo "- repo_pb2_grpc.py (service definitions)"
echo "Done!"
