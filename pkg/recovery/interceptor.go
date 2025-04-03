package recovery

import (
	"context"
	"fmt"
	"google.golang.org/grpc"
	"runtime/debug"
)

type PanicError struct {
	Panic any
	Stack []byte
}

func (p *PanicError) Error() string {
	return fmt.Sprintf("panic occured: %v", p.Panic)
}

func RecoverInterceptor(
	ctx context.Context,
	req interface{},
	_ *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = &PanicError{
				Panic: r,
				Stack: debug.Stack(),
			}
		}
	}()

	return handler(ctx, req)
}
