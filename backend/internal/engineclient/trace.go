package engineclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// KunciTrace adalah nama metadata gRPC yang membawa nomor rangkaian log.
const KunciTrace = "x-ila-trace"

// traceInterceptor menyalin trace dari konteks ke metadata gRPC, sehingga log
// Runner dan log Engine memakai nomor yang sama untuk satu alur.
//
// Tanpa ini, menelusuri satu sinyal yang lahir di VPS 1 dan dieksekusi di
// VPS 2 harus dilakukan dengan menebak waktu.
func traceInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any,
		cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		ctx = sisipkanTrace(ctx)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}

// traceStreamInterceptor sama dengan traceInterceptor untuk panggilan streaming.
func traceStreamInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn,
		method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		ctx = sisipkanTrace(ctx)
		return streamer(ctx, desc, cc, method, opts...)
	}
}

// sisipkanTrace menambahkan trace ke metadata bila konteks memilikinya.
func sisipkanTrace(ctx context.Context) context.Context {
	trace := TraceDariContext(ctx)
	if trace == "" {
		return ctx
	}
	return metadata.AppendToOutgoingContext(ctx, KunciTrace, trace)
}

type kunciTrace struct{}

// DenganTrace menempelkan trace pada konteks agar ikut terkirim ke Engine.
func DenganTrace(ctx context.Context, trace string) context.Context {
	return context.WithValue(ctx, kunciTrace{}, trace)
}

// TraceDariContext mengambil trace dari konteks; kosong bila tidak ada.
func TraceDariContext(ctx context.Context) string {
	if v, ok := ctx.Value(kunciTrace{}).(string); ok {
		return v
	}
	return ""
}
