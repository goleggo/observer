package server

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// StatsHandler returns gRPC server options that instrument
// all RPCs with OpenTelemetry traces and metrics.
func StatsHandler() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
	}
}
