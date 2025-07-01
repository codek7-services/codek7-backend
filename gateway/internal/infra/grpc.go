package infra

import (
    "context"
    "log"
    "net"
    "os"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
)

func MakeGRPCClientConn() *grpc.ClientConn {
    addr := os.Getenv("REPO_SERVICE_ADDR")
    if addr == "" {
        addr = "localhost:50051"
    }

    conn, err := grpc.NewClient(
        addr,
        grpc.WithTransportCredentials(insecure.NewCredentials()), // swap for TLS creds in prod
        grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
            dialer := &net.Dialer{}
            return dialer.DialContext(ctx, "tcp", s)
        }),
    )
    if err != nil {
        log.Fatalf("failed to connect to Repo Service at %s: %v", addr, err)
    }
    return conn
}
