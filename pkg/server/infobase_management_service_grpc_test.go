package server

import (
	"context"
	"testing"

	clientv1 "github.com/v8platform/protos/gen/ras/client/v1"
	messagesv1 "github.com/v8platform/protos/gen/ras/messages/v1"
	serializev1 "github.com/v8platform/protos/gen/v8platform/serialize/v1"
	pb "github.com/v8platform/ras-grpc-gw/pkg/gen/infobase/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

// ==================== CreateInfobase Tests ====================

func TestCreateInfobase_Success(t *testing.T) {
	// Mock: База не существует (GetShortInfobases возвращает пустой список)
	// Затем успешно создается
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return &MockEndpoint{
				RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
					// Первый вызов: GetShortInfobases (для проверки существования)
					// Проверяем тип request
					var getInfobasesReq messagesv1.GetInfobasesShortRequest
					if err := req.Request.UnmarshalTo(&getInfobasesReq); err == nil {
						// Это GetShortInfobases - возвращаем пустой список (база не найдена)
						response := &messagesv1.GetInfobasesShortResponse{
							Sessions: []*serializev1.InfobaseSummaryInfo{},
						}
						return anypb.New(response)
					}

					// Второй вызов: CreateInfobase
					// Возвращаем созданную базу
					infobaseInfo := &serializev1.InfobaseInfo{
						Uuid:      "test-uuid-123",
						Name:      "TestBase",
						ClusterId: "cluster-123",
						Dbms:      "PostgreSQL",
						DbServer:  "localhost",
						DbName:    "testdb",
					}
					return anypb.New(infobaseInfo)
				},
			}, nil
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	req := &pb.CreateInfobaseRequest{
		ClusterId: "cluster-123",
		Name:      "TestBase",
		Dbms:      pb.DBMSType_DBMS_TYPE_POSTGRESQL,
		DbServer:  "localhost",
		DbName:    "testdb",
	}

	resp, err := server.CreateInfobase(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "test-uuid-123", resp.InfobaseId)
	assert.Equal(t, "TestBase", resp.Name)
	assert.Contains(t, resp.Message, "created successfully")
}

func TestCreateInfobase_GetEndpointError(t *testing.T) {
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return nil, status.Error(codes.Unavailable, "RAS server unavailable")
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	req := &pb.CreateInfobaseRequest{
		ClusterId: "cluster-123",
		Name:      "TestBase",
		Dbms:      pb.DBMSType_DBMS_TYPE_POSTGRESQL,
		DbServer:  "localhost",
		DbName:    "testdb",
	}

	resp, err := server.CreateInfobase(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestCreateInfobase_RequestError(t *testing.T) {
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return &MockEndpoint{
				RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
					// Первый вызов (GetShortInfobases) - успех, база не найдена
					var getInfobasesReq messagesv1.GetInfobasesShortRequest
					if err := req.Request.UnmarshalTo(&getInfobasesReq); err == nil {
						response := &messagesv1.GetInfobasesShortResponse{
							Sessions: []*serializev1.InfobaseSummaryInfo{},
						}
						return anypb.New(response)
					}

					// Второй вызов (CreateInfobase) - ошибка доступа
					return nil, status.Error(codes.PermissionDenied, "access denied")
				},
			}, nil
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	req := &pb.CreateInfobaseRequest{
		ClusterId: "cluster-123",
		Name:      "TestBase",
		Dbms:      pb.DBMSType_DBMS_TYPE_POSTGRESQL,
		DbServer:  "localhost",
		DbName:    "testdb",
	}

	resp, err := server.CreateInfobase(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestCreateInfobase_Idempotent(t *testing.T) {
	// Mock: База уже существует
	existingUUID := "existing-uuid-456"
	existingName := "TestBase"

	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return &MockEndpoint{
				RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
					// GetShortInfobases - возвращаем существующую базу
					var getInfobasesReq messagesv1.GetInfobasesShortRequest
					if err := req.Request.UnmarshalTo(&getInfobasesReq); err == nil {
						response := &messagesv1.GetInfobasesShortResponse{
							Sessions: []*serializev1.InfobaseSummaryInfo{
								{
									Uuid: existingUUID,
									Name: existingName,
								},
							},
						}
						return anypb.New(response)
					}

					// Не должны дойти до CreateInfobase
					t.Error("CreateInfobase should not be called for existing infobase")
					return nil, status.Error(codes.Internal, "unexpected call")
				},
			}, nil
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	req := &pb.CreateInfobaseRequest{
		ClusterId: "cluster-123",
		Name:      existingName,
		Dbms:      pb.DBMSType_DBMS_TYPE_POSTGRESQL,
		DbServer:  "localhost",
		DbName:    "testdb",
	}

	resp, err := server.CreateInfobase(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, existingUUID, resp.InfobaseId)
	assert.Equal(t, existingName, resp.Name)
	assert.Contains(t, resp.Message, "already exists")
}

func TestCreateInfobase_AllOptionalFields(t *testing.T) {
	// Test с всеми optional полями для максимального coverage
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return &MockEndpoint{
				RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
					// Первый вызов: GetShortInfobases (база не найдена)
					var getInfobasesReq messagesv1.GetInfobasesShortRequest
					if err := req.Request.UnmarshalTo(&getInfobasesReq); err == nil {
						response := &messagesv1.GetInfobasesShortResponse{
							Sessions: []*serializev1.InfobaseSummaryInfo{},
						}
						return anypb.New(response)
					}

					// Второй вызов: CreateInfobase - успех
					infobaseInfo := &serializev1.InfobaseInfo{
						Uuid:      "test-uuid-456",
						Name:      "CompleteBase",
						ClusterId: "cluster-123",
					}
					return anypb.New(infobaseInfo)
				},
			}, nil
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	// Helper для создания указателей
	strPtr := func(s string) *string { return &s }
	int32Ptr := func(i int32) *int32 { return &i }
	boolPtr := func(b bool) *bool { return &b }
	secLevelPtr := func(l pb.SecurityLevel) *pb.SecurityLevel { return &l }

	req := &pb.CreateInfobaseRequest{
		ClusterId:                 "cluster-123",
		Name:                      "CompleteBase",
		Dbms:                      pb.DBMSType_DBMS_TYPE_POSTGRESQL,
		DbServer:                  "localhost",
		DbName:                    "testdb",
		DbUser:                    strPtr("testuser"),
		DbPassword:                strPtr("testpass"),
		Locale:                    strPtr("en_US"),
		DateOffset:                int32Ptr(3),
		Description:               strPtr("Test database"),
		SecurityLevel:             secLevelPtr(pb.SecurityLevel_SECURITY_LEVEL_1),
		ScheduledJobsDeny:         boolPtr(false),
		LicenseDistributionAllow:  boolPtr(true),
		ClusterUser:               strPtr("admin"),
		ClusterPassword:           strPtr("admin123"),
	}

	resp, err := server.CreateInfobase(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "test-uuid-456", resp.InfobaseId)
	assert.Equal(t, "CompleteBase", resp.Name)
}

func TestCreateInfobase_ValidationErrors(t *testing.T) {
	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: &MockRASClient{}, // не будет вызван
	}

	tests := []struct {
		name    string
		req     *pb.CreateInfobaseRequest
		wantErr codes.Code
	}{
		{
			name: "empty cluster_id",
			req: &pb.CreateInfobaseRequest{
				ClusterId: "",
				Name:      "TestBase",
				Dbms:      pb.DBMSType_DBMS_TYPE_POSTGRESQL,
				DbServer:  "localhost",
				DbName:    "testdb",
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "empty name",
			req: &pb.CreateInfobaseRequest{
				ClusterId: "cluster-123",
				Name:      "",
				Dbms:      pb.DBMSType_DBMS_TYPE_POSTGRESQL,
				DbServer:  "localhost",
				DbName:    "testdb",
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "invalid dbms",
			req: &pb.CreateInfobaseRequest{
				ClusterId: "cluster-123",
				Name:      "TestBase",
				Dbms:      pb.DBMSType_DBMS_TYPE_UNSPECIFIED,
				DbServer:  "localhost",
				DbName:    "testdb",
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "missing db_server",
			req: &pb.CreateInfobaseRequest{
				ClusterId: "cluster-123",
				Name:      "TestBase",
				Dbms:      pb.DBMSType_DBMS_TYPE_POSTGRESQL,
				DbServer:  "",
				DbName:    "testdb",
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "missing db_name",
			req: &pb.CreateInfobaseRequest{
				ClusterId: "cluster-123",
				Name:      "TestBase",
				Dbms:      pb.DBMSType_DBMS_TYPE_POSTGRESQL,
				DbServer:  "localhost",
				DbName:    "",
			},
			wantErr: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := server.CreateInfobase(context.Background(), tt.req)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, tt.wantErr, status.Code(err))
		})
	}
}

// ==================== UpdateInfobase Tests ====================

func TestUpdateInfobase_Success(t *testing.T) {
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return &MockEndpoint{
				RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
					// Возвращаем обновленную базу
					updatedInfobase := &serializev1.InfobaseInfo{
						Uuid:          "infobase-123",
						ClusterId:     "cluster-123",
						SessionsDeny:  true,
						DeniedMessage: "Maintenance",
					}
					return anypb.New(updatedInfobase)
				},
			}, nil
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	sessionsDeny := true
	deniedMsg := "Maintenance"

	req := &pb.UpdateInfobaseRequest{
		ClusterId:     "cluster-123",
		InfobaseId:    "infobase-123",
		SessionsDeny:  &sessionsDeny,
		DeniedMessage: &deniedMsg,
	}

	resp, err := server.UpdateInfobase(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "infobase-123", resp.InfobaseId)
	assert.True(t, resp.Success)
}

func TestUpdateInfobase_AllOptionalFields(t *testing.T) {
	// Test с всеми optional полями для максимального coverage
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return &MockEndpoint{
				RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
					updatedInfobase := &serializev1.InfobaseInfo{
						Uuid:      "infobase-123",
						ClusterId: "cluster-123",
					}
					return anypb.New(updatedInfobase)
				},
			}, nil
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	// Все optional поля
	sessionsDeny := true
	scheduledJobsDeny := false
	deniedMsg := "Test message"
	permissionCode := "12345"
	dbms := pb.DBMSType_DBMS_TYPE_POSTGRESQL
	dbServer := "new-server"
	dbName := "new-db"
	dbUser := "new-user"
	dbPassword := "new-pass"
	description := "Updated description"
	securityLevel := pb.SecurityLevel_SECURITY_LEVEL_2

	req := &pb.UpdateInfobaseRequest{
		ClusterId:         "cluster-123",
		InfobaseId:        "infobase-123",
		SessionsDeny:      &sessionsDeny,
		ScheduledJobsDeny: &scheduledJobsDeny,
		DeniedMessage:     &deniedMsg,
		PermissionCode:    &permissionCode,
		Dbms:              &dbms,
		DbServer:          &dbServer,
		DbName:            &dbName,
		DbUser:            &dbUser,
		DbPassword:        &dbPassword,
		Description:       &description,
		SecurityLevel:     &securityLevel,
	}

	resp, err := server.UpdateInfobase(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestUpdateInfobase_GetEndpointError(t *testing.T) {
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return nil, status.Error(codes.Unavailable, "RAS unavailable")
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	req := &pb.UpdateInfobaseRequest{
		ClusterId:  "cluster-123",
		InfobaseId: "infobase-123",
	}

	resp, err := server.UpdateInfobase(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestUpdateInfobase_RequestError(t *testing.T) {
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return &MockEndpoint{
				RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
					return nil, status.Error(codes.NotFound, "infobase not found")
				},
			}, nil
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	req := &pb.UpdateInfobaseRequest{
		ClusterId:  "cluster-123",
		InfobaseId: "infobase-123",
	}

	resp, err := server.UpdateInfobase(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

func TestUpdateInfobase_ValidationErrors(t *testing.T) {
	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: &MockRASClient{},
	}

	tests := []struct {
		name    string
		req     *pb.UpdateInfobaseRequest
		wantErr codes.Code
	}{
		{
			name: "empty cluster_id",
			req: &pb.UpdateInfobaseRequest{
				ClusterId:  "",
				InfobaseId: "infobase-123",
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "empty infobase_id",
			req: &pb.UpdateInfobaseRequest{
				ClusterId:  "cluster-123",
				InfobaseId: "",
			},
			wantErr: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := server.UpdateInfobase(context.Background(), tt.req)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, tt.wantErr, status.Code(err))
		})
	}
}

// ==================== DropInfobase Tests ====================

func TestDropInfobase_UnregisterOnly_Success(t *testing.T) {
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return &MockEndpoint{
				RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
					// Возвращаем успешный ответ
					infobaseInfo := &serializev1.InfobaseInfo{
						Uuid:      "infobase-123",
						ClusterId: "cluster-123",
					}
					return anypb.New(infobaseInfo)
				},
			}, nil
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	req := &pb.DropInfobaseRequest{
		ClusterId:  "cluster-123",
		InfobaseId: "infobase-123",
		DropMode:   pb.DropMode_DROP_MODE_UNREGISTER_ONLY,
	}

	resp, err := server.DropInfobase(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "infobase-123", resp.InfobaseId)
	assert.True(t, resp.Success)
	assert.Contains(t, resp.Message, "dropped successfully")
}

func TestDropInfobase_UnsupportedDropMode(t *testing.T) {
	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: &MockRASClient{}, // не будет вызван
	}

	tests := []struct {
		name     string
		dropMode pb.DropMode
	}{
		{"DROP_DATABASE", pb.DropMode_DROP_MODE_DROP_DATABASE},
		{"CLEAR_DATABASE", pb.DropMode_DROP_MODE_CLEAR_DATABASE},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &pb.DropInfobaseRequest{
				ClusterId:  "cluster-123",
				InfobaseId: "infobase-123",
				DropMode:   tt.dropMode,
			}

			resp, err := server.DropInfobase(context.Background(), req)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, codes.Unimplemented, status.Code(err))
			assert.Contains(t, err.Error(), "not supported by RAS Binary Protocol")
		})
	}
}

func TestDropInfobase_ValidationErrors(t *testing.T) {
	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: &MockRASClient{},
	}

	tests := []struct {
		name    string
		req     *pb.DropInfobaseRequest
		wantErr codes.Code
	}{
		{
			name: "empty cluster_id",
			req: &pb.DropInfobaseRequest{
				ClusterId:  "",
				InfobaseId: "infobase-123",
				DropMode:   pb.DropMode_DROP_MODE_UNREGISTER_ONLY,
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "empty infobase_id",
			req: &pb.DropInfobaseRequest{
				ClusterId:  "cluster-123",
				InfobaseId: "",
				DropMode:   pb.DropMode_DROP_MODE_UNREGISTER_ONLY,
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "unspecified drop_mode",
			req: &pb.DropInfobaseRequest{
				ClusterId:  "cluster-123",
				InfobaseId: "infobase-123",
				DropMode:   pb.DropMode_DROP_MODE_UNSPECIFIED,
			},
			wantErr: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := server.DropInfobase(context.Background(), tt.req)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, tt.wantErr, status.Code(err))
		})
	}
}

func TestDropInfobase_GetEndpointError(t *testing.T) {
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return nil, status.Error(codes.Unavailable, "RAS unavailable")
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	req := &pb.DropInfobaseRequest{
		ClusterId:  "cluster-123",
		InfobaseId: "infobase-123",
		DropMode:   pb.DropMode_DROP_MODE_UNREGISTER_ONLY,
	}

	resp, err := server.DropInfobase(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}

func TestDropInfobase_RequestError(t *testing.T) {
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return &MockEndpoint{
				RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
					return nil, status.Error(codes.NotFound, "infobase not found")
				},
			}, nil
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	req := &pb.DropInfobaseRequest{
		ClusterId:  "cluster-123",
		InfobaseId: "infobase-123",
		DropMode:   pb.DropMode_DROP_MODE_UNREGISTER_ONLY,
	}

	resp, err := server.DropInfobase(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, resp)
	assert.Equal(t, codes.NotFound, status.Code(err))
}

// ==================== LockInfobase Tests ====================

func TestLockInfobase_Success(t *testing.T) {
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return &MockEndpoint{
				RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
					// UpdateInfobase успешно выполняется
					infobaseInfo := &serializev1.InfobaseInfo{
						Uuid:         "infobase-123",
						SessionsDeny: true,
					}
					return anypb.New(infobaseInfo)
				},
			}, nil
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	req := &pb.LockInfobaseRequest{
		ClusterId:         "cluster-123",
		InfobaseId:        "infobase-123",
		SessionsDeny:      true,
		ScheduledJobsDeny: false,
	}

	resp, err := server.LockInfobase(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "infobase-123", resp.InfobaseId)
	assert.True(t, resp.Success)
}

func TestLockInfobase_ValidationErrors(t *testing.T) {
	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: &MockRASClient{},
	}

	tests := []struct {
		name    string
		req     *pb.LockInfobaseRequest
		wantErr codes.Code
	}{
		{
			name: "empty cluster_id",
			req: &pb.LockInfobaseRequest{
				ClusterId:  "",
				InfobaseId: "infobase-123",
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "empty infobase_id",
			req: &pb.LockInfobaseRequest{
				ClusterId:  "cluster-123",
				InfobaseId: "",
			},
			wantErr: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := server.LockInfobase(context.Background(), tt.req)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, tt.wantErr, status.Code(err))
		})
	}
}

// ==================== UnlockInfobase Tests ====================

func TestUnlockInfobase_Success(t *testing.T) {
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return &MockEndpoint{
				RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
					// UpdateInfobase успешно выполняется
					infobaseInfo := &serializev1.InfobaseInfo{
						Uuid:         "infobase-123",
						SessionsDeny: false,
					}
					return anypb.New(infobaseInfo)
				},
			}, nil
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	req := &pb.UnlockInfobaseRequest{
		ClusterId:           "cluster-123",
		InfobaseId:          "infobase-123",
		UnlockSessions:      true,
		UnlockScheduledJobs: true,
	}

	resp, err := server.UnlockInfobase(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "infobase-123", resp.InfobaseId)
	assert.True(t, resp.Success)
}

func TestUnlockInfobase_OnlySessions(t *testing.T) {
	// Test unlock только sessions (не scheduled_jobs)
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return &MockEndpoint{
				RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
					infobaseInfo := &serializev1.InfobaseInfo{
						Uuid:         "infobase-123",
						SessionsDeny: false,
					}
					return anypb.New(infobaseInfo)
				},
			}, nil
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	req := &pb.UnlockInfobaseRequest{
		ClusterId:           "cluster-123",
		InfobaseId:          "infobase-123",
		UnlockSessions:      true,
		UnlockScheduledJobs: false,
	}

	resp, err := server.UnlockInfobase(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestUnlockInfobase_OnlyScheduledJobs(t *testing.T) {
	// Test unlock только scheduled_jobs (не sessions)
	mockClient := &MockRASClient{
		GetEndpointFunc: func(ctx context.Context) (clientv1.EndpointServiceImpl, error) {
			return &MockEndpoint{
				RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
					infobaseInfo := &serializev1.InfobaseInfo{
						Uuid:              "infobase-123",
						ScheduledJobsDeny: false,
					}
					return anypb.New(infobaseInfo)
				},
			}, nil
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: mockClient,
	}

	req := &pb.UnlockInfobaseRequest{
		ClusterId:           "cluster-123",
		InfobaseId:          "infobase-123",
		UnlockSessions:      false,
		UnlockScheduledJobs: true,
	}

	resp, err := server.UnlockInfobase(context.Background(), req)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestUnlockInfobase_ValidationErrors(t *testing.T) {
	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: &MockRASClient{},
	}

	tests := []struct {
		name    string
		req     *pb.UnlockInfobaseRequest
		wantErr codes.Code
	}{
		{
			name: "empty cluster_id",
			req: &pb.UnlockInfobaseRequest{
				ClusterId:  "",
				InfobaseId: "infobase-123",
			},
			wantErr: codes.InvalidArgument,
		},
		{
			name: "empty infobase_id",
			req: &pb.UnlockInfobaseRequest{
				ClusterId:  "cluster-123",
				InfobaseId: "",
			},
			wantErr: codes.InvalidArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := server.UnlockInfobase(context.Background(), tt.req)

			assert.Error(t, err)
			assert.Nil(t, resp)
			assert.Equal(t, tt.wantErr, status.Code(err))
		})
	}
}

// ==================== findInfobaseByName Tests ====================

func TestFindInfobaseByName_Found(t *testing.T) {
	expectedUUID := "found-uuid-789"
	expectedName := "FoundBase"

	mockEndpoint := &MockEndpoint{
		RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
			// GetShortInfobases - возвращаем базу
			response := &messagesv1.GetInfobasesShortResponse{
				Sessions: []*serializev1.InfobaseSummaryInfo{
					{
						Uuid: expectedUUID,
						Name: expectedName,
					},
					{
						Uuid: "other-uuid",
						Name: "OtherBase",
					},
				},
			}
			return anypb.New(response)
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: &MockRASClient{},
	}

	result, err := server.findInfobaseByName(
		context.Background(),
		mockEndpoint,
		"cluster-123",
		expectedName,
	)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedUUID, result.GetUuid())
	assert.Equal(t, expectedName, result.GetName())
}

func TestFindInfobaseByName_NotFound(t *testing.T) {
	mockEndpoint := &MockEndpoint{
		RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
			// GetShortInfobases - возвращаем пустой список
			response := &messagesv1.GetInfobasesShortResponse{
				Sessions: []*serializev1.InfobaseSummaryInfo{},
			}
			return anypb.New(response)
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: &MockRASClient{},
	}

	result, err := server.findInfobaseByName(
		context.Background(),
		mockEndpoint,
		"cluster-123",
		"NonExistentBase",
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, codes.NotFound, status.Code(err))
	assert.Contains(t, err.Error(), "not found")
}

func TestFindInfobaseByName_RASError(t *testing.T) {
	mockEndpoint := &MockEndpoint{
		RequestFunc: func(ctx context.Context, req *clientv1.EndpointRequest) (*anypb.Any, error) {
			// RAS error при GetShortInfobases
			return nil, status.Error(codes.Unavailable, "RAS unavailable")
		},
	}

	server := &InfobaseManagementServer{
		logger: zap.NewNop(),
		client: &MockRASClient{},
	}

	result, err := server.findInfobaseByName(
		context.Background(),
		mockEndpoint,
		"cluster-123",
		"TestBase",
	)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Equal(t, codes.Unavailable, status.Code(err))
}
