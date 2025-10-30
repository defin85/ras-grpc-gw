package server

import (
	"context"
	"fmt"
	"net"
	"time"

	_ "github.com/lithammer/shortuuid/v3"
	"github.com/spf13/cast"
	clientv1 "github.com/v8platform/protos/gen/ras/client/v1"
	messagesv1 "github.com/v8platform/protos/gen/ras/messages/v1"
	ras_service "github.com/v8platform/protos/gen/ras/service/api/v1"
	"github.com/v8platform/ras-grpc-gw/pkg/client"
	access_service "github.com/v8platform/ras-grpc-gw/pkg/gen/access/service"
	"github.com/v8platform/ras-grpc-gw/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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
	
	// Сохраняем ссылку на сервер
	s.grpcServer = grpc.NewServer()
	
	ras_service.RegisterAuthServiceServer(s.grpcServer, srv)
	ras_service.RegisterClustersServiceServer(s.grpcServer, srv)
	ras_service.RegisterSessionsServiceServer(s.grpcServer, srv)
	ras_service.RegisterInfobasesServiceServer(s.grpcServer, srv)

	accessSrv := NewAccessServer()

	access_service.RegisterClientServiceServer(s.grpcServer, accessSrv)
	access_service.RegisterTokenServiceServer(s.grpcServer, accessSrv)

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
	endpoint, err := s.client.GetEndpoint(ctx)
	if err != nil {
		return nil, err
	}
	auth := clientv1.NewAuthService(endpoint)

	return auth.AuthenticateCluster(ctx, request)

}

func (s *rasClientServiceServer) withEndpoint(ctx context.Context, fn func(clientv1.EndpointServiceImpl) error) (err error) {
	var endpoint clientv1.EndpointServiceImpl
	endpoint, err = s.client.GetEndpoint(ctx)

	if err != nil {
		return err
	}

	defer func() {
		if err == nil {
			header := metadata.New(map[string]string{
				"endpoint_id": cast.ToString(endpoint),
				//"host": cast.ToString(s.client.),
			})

			_ = grpc.SendHeader(ctx, header)
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
