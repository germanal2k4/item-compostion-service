package server

import (
	"cmp"
	"context"
	"fmt"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"item_compositiom_service/pkg/logger"
	"net"
	"os"
	"os/user"
	"strconv"
	"strings"
	"sync"

	servicepb "item_compositiom_service/internal/generated/service"
	"item_compositiom_service/internal/services"
)

type Server struct {
	config       *Config
	server       *grpc.Server
	wg           sync.WaitGroup
	resErr       error
	runCtx       context.Context
	runCancelFn  context.CancelFunc
	healthServer *health.Server
}

func NewServer(
	lc fx.Lifecycle,
	config *Config,

	implItemCompositionService *services.Service,

	lgrInterceptor *logger.Interceptor,
) (*Server, error) {
	if config == nil {
		return nil, fmt.Errorf("gRPC server config is nil")
	}

	if config.ListenAddress == "" {
		return nil, fmt.Errorf("gRPC server listen address is empty")
	}

	var logOpts []logger.LogOption

	if config.Logging != nil {
		logOpts = append(logOpts,
			logger.WithDisable(config.Logging.Disable),
			logger.WithDisableEnrichTraces(config.Logging.DisableEnrichTraces),
			logger.WithDisableLogRequest(config.Logging.DisableLogRequestMessage),
			logger.WithDisableLogResponse(config.Logging.DisableLogResponseMessage),
		)
	}

	if config.Logging != nil && config.Logging.MaxMessageSize != 0 {
		logOpts = append(logOpts, logger.WithMaxMessageSize(config.Logging.MaxMessageSize))
	}

	server := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			lgrInterceptor.GetServerInterceptor(logOpts...),
			// TODO set metrics, trace, recover, deadline, health interceptors
		),
	)

	servicepb.RegisterItemCompositionServiceServer(server, implItemCompositionService)

	reflection.Register(server)

	healthServer := health.NewServer()
	healthgrpc.RegisterHealthServer(server, healthServer)

	ctx, cancel := context.WithCancel(context.Background())

	res := &Server{
		config:       config,
		server:       server,
		runCtx:       ctx,
		runCancelFn:  cancel,
		healthServer: healthServer,
	}

	res.healthServer.SetServingStatus("", healthgrpc.HealthCheckResponse_NOT_SERVING)

	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			return res.Start(ctx)
		},
		OnStop: func(ctx context.Context) error {
			return res.Stop(ctx)
		},
	})

	return res, nil
}

func (s *Server) GRPCServerDescriptor() {}

func (s *Server) Start(_ context.Context) error {
	lis, err := s.listen()
	if err != nil {
		return fmt.Errorf("gRPC server listen: %w", err)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		s.resErr = s.server.Serve(lis)
	}()

	// TODO maybe add health check?

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.runCancelFn()
		s.server.GracefulStop()
		s.wg.Wait()
	}()

	select {
	case <-ctx.Done():
		return fmt.Errorf("stop gRPC server: %w", ctx.Err())
	case <-done:
	}

	return s.resErr
}

func (s *Server) listen() (net.Listener, error) {
	var (
		addr    = s.config.ListenAddress
		network = "tcp"
	)

	if strings.HasPrefix(addr, "unix:") {
		addr = strings.TrimPrefix(addr, "unix:")
		network = "unix"

		err := os.RemoveAll(addr)
		if err != nil {
			return nil, fmt.Errorf("remove existing socket file: %w", err)
		}
	}

	lis, err := net.Listen(network, addr)
	if err != nil {
		return nil, fmt.Errorf("gRPC server net listen: %w", err)
	}

	if network == "unix" {
		if err := s.updateUnixSocketPermissions(addr); err != nil {
			return nil, fmt.Errorf("update unix socket permissions: %s: %w", addr, err)
		}
	}

	return lis, nil
}

func (s *Server) updateUnixSocketPermissions(path string) error {
	if err := os.Chmod(path, 0660); err != nil {
		return fmt.Errorf("change socket file mode: %s: %w", path, err)
	}

	username := cmp.Or(s.config.UnixSocketUser, "www-data")

	u, err := user.Lookup(username)
	if err != nil {
		return fmt.Errorf("lookup user %s: %w", username, err)
	}

	uid, err := strconv.Atoi(u.Uid)
	if err != nil {
		return fmt.Errorf("convert uid to int: %s: %w", u.Uid, err)
	}

	gid, err := strconv.Atoi(u.Gid)
	if err != nil {
		return fmt.Errorf("convert gid to int: %s: %w", u.Gid, err)
	}

	err = os.Chown(path, uid, gid)
	if err != nil {
		return fmt.Errorf("change socket file owner: %s: %w", path, err)
	}

	return nil
}
