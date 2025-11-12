package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	_ "github.com/lithammer/shortuuid/v3"
	"github.com/spf13/cast"
	clientv1 "github.com/v8platform/protos/gen/ras/client/v1"
	messagesv1 "github.com/v8platform/protos/gen/ras/messages/v1"
	protocolv1 "github.com/v8platform/protos/gen/ras/protocol/v1"
	ras_service "github.com/v8platform/protos/gen/ras/service/api/v1"
	serializev1 "github.com/v8platform/protos/gen/v8platform/serialize/v1"
	"github.com/v8platform/ras-grpc-gw/pkg/client"
	access_service "github.com/v8platform/ras-grpc-gw/pkg/gen/access/service"
	infobase_service "github.com/v8platform/ras-grpc-gw/pkg/gen/infobase/service"
	"github.com/v8platform/ras-grpc-gw/pkg/logger"
"github.com/v8platform/ras-grpc-gw/pkg/interceptor"
	"github.com/v8platform/ras-grpc-gw/pkg/tlsconfig"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
)

func NewRASServer(rasAddr string) *RASServer {
	return &RASServer{
		rasAddr: rasAddr,
	}
}

type RASServer struct {
	rasAddr    string
	grpcServer *grpc.Server
	rasService *rasClientServiceServer // Added for HTTP handler access

	idxClients   map[string]*ClientInfo
	idxEndpoints map[string]*EndpointInfo
}

type EndpointInfo struct {
	uuid       string
	client     *ClientInfo
	EndpointId string
}

type ClientInfo struct {
	uuid        string
	conn        *client.ClientConn
	IdleTimeout time.Duration
}

func (s *RASServer) Serve(host string) error {

	listener, err := net.Listen("tcp", host)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", host, err)
	}

	srv := NewRasClientServiceServer(s.rasAddr)
	// Store for HTTP handler access (type assertion)
	if rasService, ok := srv.(*rasClientServiceServer); ok {
		s.rasService = rasService
	}

	// Load TLS configuration
	tlsConfig, err := tlsconfig.LoadTLSConfig(logger.Log)
	if err != nil {
		return fmt.Errorf("failed to load TLS config: %w", err)
	}

	// Setup gRPC server options with interceptors
	var opts []grpc.ServerOption

	// Add interceptors
	opts = append(opts, grpc.ChainUnaryInterceptor(
		interceptor.SanitizePasswordsInterceptor(logger.Log),
		interceptor.AuditInterceptor(logger.Log),
	))

	// Add TLS if enabled
	if tlsConfig != nil {
		creds := credentials.NewTLS(tlsConfig)
		opts = append(opts, grpc.Creds(creds))
		logger.Log.Info("TLS enabled for gRPC server")
	} else {
		logger.Log.Warn("TLS disabled - passwords transmitted in plaintext! Enable with TLS_ENABLED=true")
	}

	// Create gRPC server with options
	s.grpcServer = grpc.NewServer(opts...)
	
	ras_service.RegisterAuthServiceServer(s.grpcServer, srv)
	ras_service.RegisterClustersServiceServer(s.grpcServer, srv)
	ras_service.RegisterSessionsServiceServer(s.grpcServer, srv)
	ras_service.RegisterInfobasesServiceServer(s.grpcServer, srv)

	accessSrv := NewAccessServer()

	access_service.RegisterClientServiceServer(s.grpcServer, accessSrv)
	access_service.RegisterTokenServiceServer(s.grpcServer, accessSrv)

	// Register InfobaseManagementService (Sprint 3.2, Day 1-2)
	rasClient := NewRASClient(s.rasAddr)
	infobaseMgmtSrv := NewInfobaseManagementServer(rasClient)
	infobase_service.RegisterInfobaseManagementServiceServer(s.grpcServer, infobaseMgmtSrv)

	logger.Log.Info("Listening on", zap.String("address", host))
	if err := s.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("failed to serve gRPC server: %w", err)
	}

	return nil
}

// GracefulStop gracefully stops the gRPC server
func (s *RASServer) GracefulStop(ctx context.Context) error {
	if s.grpcServer != nil {
		// Создаем канал для отслеживания завершения
		stopped := make(chan struct{})

		go func() {
			s.grpcServer.GracefulStop()
			close(stopped)
		}()

		// Ждем завершения или таймаута
		select {
		case <-stopped:
			logger.Log.Info("gRPC server stopped gracefully")
			return nil
		case <-ctx.Done():
			// Если таймаут - форсируем остановку
			logger.Log.Warn("Graceful shutdown timeout, forcing stop")
			s.grpcServer.Stop()
			return ctx.Err()
		}
	}
	return nil
}

// Check проверяет подключение к RAS серверу
func (s *RASServer) Check(ctx context.Context) error {
	// TODO: Реализовать реальную проверку подключения к RAS
	// Пока что заглушка, которая всегда возвращает успех

	// В будущем здесь должна быть проверка:
	// - Подключение к RAS серверу доступно
	// - Можно выполнить простой запрос (например, получить список кластеров)

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// Проверка что RAS адрес установлен
		if s.rasAddr == "" {
			return fmt.Errorf("RAS address not configured")
		}
		return nil
	}
}

// GetTerminateSessionHandler returns HTTP handler for TerminateSession endpoint
// This allows health server to expose /api/v1/sessions/terminate for HTTP clients
func (s *RASServer) GetTerminateSessionHandler() http.HandlerFunc {
	if s.rasService == nil {
		// Return error handler if service not initialized
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "RAS service not initialized", http.StatusServiceUnavailable)
		}
	}
	return s.rasService.HandleTerminateSession
}

func NewRasClientServiceServer(rasAddr string) ras_service.RASServiceServer {
	return &rasClientServiceServer{
		client: client.NewClientConn(rasAddr),
	}
}

type rasClientServiceServer struct {
	ras_service.UnimplementedRASServiceServer
	client *client.ClientConn
}

func (s *rasClientServiceServer) AuthenticateCluster(ctx context.Context, request *messagesv1.ClusterAuthenticateRequest) (*emptypb.Empty, error) {

	var resp *emptypb.Empty
	var err error

	err = s.withEndpoint(ctx, func(endpoint clientv1.EndpointServiceImpl) error {
		auth := clientv1.NewAuthService(endpoint)
		resp, err = auth.AuthenticateCluster(ctx, request)
		if err != nil {
			return err
		}
		return nil

	})

	return resp, err
}

func (s *rasClientServiceServer) withEndpoint(ctx context.Context, fn func(clientv1.EndpointServiceImpl) error) (err error) {
	var endpoint clientv1.EndpointServiceImpl
	endpoint, err = s.client.GetEndpoint(ctx)

	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			var endpointID string

			// Type assertion для извлечения endpoint ID
			// endpointService встраивает protocolv1.EndpointImpl через embedded field
			if endpointImpl, ok := endpoint.(protocolv1.EndpointImpl); ok {
				endpointID = cast.ToString(endpointImpl.GetId())
				log.Printf("[withEndpoint] Sending endpoint_id in headers: %s (extracted via type assertion)", endpointID)
			} else {
				log.Printf("[withEndpoint] WARNING: Cannot extract endpoint ID (type assertion to protocolv1.EndpointImpl failed)")
			}

			if endpointID != "" {
				header := metadata.New(map[string]string{
					"endpoint_id": endpointID,
				})
				_ = grpc.SendHeader(ctx, header)
			}
		}
	}()

	return fn(endpoint)
}

func (s *rasClientServiceServer) AuthenticateInfobase(ctx context.Context, request *messagesv1.AuthenticateInfobaseRequest) (*emptypb.Empty, error) {

	var resp *emptypb.Empty
	var err error

	err = s.withEndpoint(ctx, func(endpoint clientv1.EndpointServiceImpl) error {
		auth := clientv1.NewAuthService(endpoint)
		resp, err = auth.AuthenticateInfobase(ctx, request)
		if err != nil {
			return err
		}
		return nil

	})

	return resp, err
}

func (s *rasClientServiceServer) AuthenticateAgent(ctx context.Context, request *messagesv1.AuthenticateAgentRequest) (*emptypb.Empty, error) {
	endpoint, err := s.client.GetEndpoint(ctx)
	if err != nil {
		return nil, err
	}
	auth := clientv1.NewAuthService(endpoint)

	return auth.AuthenticateAgent(ctx, request)
}

func (s *rasClientServiceServer) GetClusters(ctx context.Context, request *messagesv1.GetClustersRequest) (*messagesv1.GetClustersResponse, error) {
	endpoint, err := s.client.GetEndpoint(ctx)
	if err != nil {
		return nil, err
	}
	service := clientv1.NewClustersService(endpoint)
	return service.GetClusters(ctx, request)
}

func (s *rasClientServiceServer) GetClusterInfo(ctx context.Context, request *messagesv1.GetClusterInfoRequest) (*messagesv1.GetClusterInfoResponse, error) {
	endpoint, err := s.client.GetEndpoint(ctx)
	if err != nil {
		return nil, err
	}
	service := clientv1.NewClustersService(endpoint)
	return service.GetClusterInfo(ctx, request)
}

func (s *rasClientServiceServer) GetSessions(ctx context.Context, request *messagesv1.GetSessionsRequest) (*messagesv1.GetSessionsResponse, error) {
	endpoint, err := s.client.GetEndpoint(ctx)
	if err != nil {
		return nil, err
	}

	service := clientv1.NewSessionsService(endpoint)
	return service.GetSessions(ctx, request)
}

func (s *rasClientServiceServer) GetShortInfobases(ctx context.Context, request *messagesv1.GetInfobasesShortRequest) (*messagesv1.GetInfobasesShortResponse, error) {
	endpoint, err := s.client.GetEndpoint(ctx)
	if err != nil {
		return nil, err
	}
	service := clientv1.NewInfobasesService(endpoint)
	return service.GetShortInfobases(ctx, request)
}

func (s *rasClientServiceServer) GetInfobaseSessions(ctx context.Context, request *messagesv1.GetInfobaseSessionsRequest) (*messagesv1.GetInfobaseSessionsResponse, error) {
	endpoint, err := s.client.GetEndpoint(ctx)
	if err != nil {
		return nil, err
	}
	service := clientv1.NewInfobasesService(endpoint)
	return service.GetSessions(ctx, request)
}

// TerminateSession terminates a session in the 1C cluster
// Implementation based on captured RAS protocol traffic analysis:
// - Message type: 0x47 (71 decimal)
// - Structure: cluster_id (UUID 16 bytes) + session_id (UUID 16 bytes)
func (s *rasClientServiceServer) TerminateSession(ctx context.Context, request *TerminateSessionRequest) (*emptypb.Empty, error) {
	logger.Log.Info("TerminateSession request",
		zap.String("cluster_id", request.ClusterId),
		zap.String("session_id", request.SessionId),
	)

	endpoint, err := s.client.GetEndpoint(ctx)
	if err != nil {
		logger.Log.Error("Failed to get RAS endpoint",
			zap.String("cluster_id", request.ClusterId),
			zap.String("session_id", request.SessionId),
			zap.Error(err),
		)
		return nil, err
	}

	// Build SessionInfo for terminate operation
	// Based on reverse-engineered protocol: only cluster_id and uuid (session_id) are required
	sessionInfo := &serializev1.SessionInfo{
		ClusterId: request.ClusterId,
		Uuid:      request.SessionId,
	}

	// Marshal to protobuf Any
	anyRequest, err := anypb.New(sessionInfo)
	if err != nil {
		logger.Log.Error("Failed to marshal TerminateSession request",
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, "failed to marshal request")
	}

	// Empty response expected (RAS returns success/failure via error)
	anyRespond, err := anypb.New(&serializev1.SessionInfo{})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create response template")
	}

	endpointReq := &clientv1.EndpointRequest{
		Request: anyRequest,
		Respond: anyRespond,
	}

	// Execute RAS request
	_, err = endpoint.Request(ctx, endpointReq)
	if err != nil {
		logger.Log.Error("Failed to terminate session via RAS",
			zap.String("cluster_id", request.ClusterId),
			zap.String("session_id", request.SessionId),
			zap.Error(err),
		)
		return nil, err
	}

	logger.Log.Info("Session terminated successfully",
		zap.String("cluster_id", request.ClusterId),
		zap.String("session_id", request.SessionId),
	)

	return &emptypb.Empty{}, nil
}
