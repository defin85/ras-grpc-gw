package server

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	clientv1 "github.com/v8platform/protos/gen/ras/client/v1"
messagesv1 "github.com/v8platform/protos/gen/ras/messages/v1"
	serializev1 "github.com/v8platform/protos/gen/v8platform/serialize/v1"
	pb "github.com/v8platform/ras-grpc-gw/pkg/gen/infobase/service"
	"github.com/v8platform/ras-grpc-gw/pkg/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// InfobaseManagementServer implements infobase management gRPC service
type InfobaseManagementServer struct {
	pb.UnimplementedInfobaseManagementServiceServer
	logger *zap.Logger
	client RASClient
}

// NewInfobaseManagementServer creates new server instance
func NewInfobaseManagementServer(client RASClient) *InfobaseManagementServer {
	return &InfobaseManagementServer{
		logger: logger.Log,
		client: client,
	}
}

// ==================== HELPER METHODS ====================
// findInfobaseByName queries RAS for an infobase with the given name
// findInfobaseByName queries RAS for an infobase with the given name
// Returns InfobaseSummaryInfo (uuid, name, descr only) if found
func (s *InfobaseManagementServer) findInfobaseByName(
	ctx context.Context,
	endpoint clientv1.EndpointServiceImpl,
	clusterID, name string,
) (*serializev1.InfobaseSummaryInfo, error) {
	// Use InfobasesService для получения списка баз
	service := clientv1.NewInfobasesService(endpoint)
	
	getInfobasesReq := &messagesv1.GetInfobasesShortRequest{
		ClusterId: clusterID,
	}
	
	response, err := service.GetShortInfobases(ctx, getInfobasesReq)
	if err != nil {
		return nil, s.mapRASError(err)
	}
	
	// Поиск базы по имени
	// response.GetSessions() возвращает список InfobaseSummaryInfo
	for _, ib := range response.GetSessions() {
		if ib.GetName() == name {
			return ib, nil
		}
	}
	
	return nil, status.Errorf(codes.NotFound, "infobase '%s' not found", name)
}


// validateClusterId проверяет что cluster_id не пустой
func (s *InfobaseManagementServer) validateClusterId(clusterId string) error {
	if strings.TrimSpace(clusterId) == "" {
		return status.Error(codes.InvalidArgument, "cluster_id is required")
	}
	return nil
}

// validateInfobaseId проверяет что infobase_id не пустой
func (s *InfobaseManagementServer) validateInfobaseId(infobaseId string) error {
	if strings.TrimSpace(infobaseId) == "" {
		return status.Error(codes.InvalidArgument, "infobase_id is required")
	}
	return nil
}

// validateName checks if the infobase name is valid.
// Valid names must:
//   - Be non-empty
//   - Be at most 64 characters
//   - Contain only Unicode letters (Latin, Cyrillic, etc.), digits, underscore, and hyphen
//
// Examples of valid names:
//   - "Accounting_2024" (Latin)
//   - "Бухгалтерия_2024" (Cyrillic)
//   - "会计_2024" (Chinese)
//   - "My-Database_123" (mixed)
//
// Examples of invalid names:
//   - "Base@123" (special characters not allowed)
//   - "My Database" (spaces not allowed)
//   - "Test🔒Base" (emoji not allowed)
func (s *InfobaseManagementServer) validateName(name string) error {
	if strings.TrimSpace(name) == "" {
		return status.Error(codes.InvalidArgument, "name is required")
	}

	// Ограничение длины (1C limit: 64 символа)
	if len(name) > 64 {
		return status.Error(codes.InvalidArgument, "name must not exceed 64 characters")
	}

	// Имя базы должно содержать только Unicode буквы, цифры, дефис и подчеркивание
	// \p{L} - любые Unicode буквы (latin, cyrillic, chinese, etc.)
	// \p{N} - любые Unicode цифры
	matched, _ := regexp.MatchString(`^[\p{L}\p{N}_-]+$`, name)
	if !matched {
		return status.Error(codes.InvalidArgument, "name must contain only letters, digits, hyphens, and underscores")
	}

	return nil
}

// validateDBMS проверяет что DBMS указан
func (s *InfobaseManagementServer) validateDBMS(dbms pb.DBMSType) error {
	if dbms == pb.DBMSType_DBMS_TYPE_UNSPECIFIED {
		return status.Error(codes.InvalidArgument, "dbms type is required")
	}
	return nil
}

// validateLockSchedule validates lock time schedule
// CRITICAL #3: Added validation for lock schedule
func (s *InfobaseManagementServer) validateLockSchedule(startTime, endTime *timestamppb.Timestamp) error {
	if startTime == nil && endTime == nil {
		// Permanent lock - OK
		return nil
	}

	if startTime == nil || endTime == nil {
		return status.Error(codes.InvalidArgument, "both start_time and end_time must be specified for scheduled lock")
	}

	start := startTime.AsTime()
	end := endTime.AsTime()
	now := time.Now()

	// end_time должен быть после start_time
	if !end.After(start) {
		return status.Error(codes.InvalidArgument, "end_time must be after start_time")
	}

	// end_time должен быть в будущем (или минимум сейчас)
	if end.Before(now) {
		return status.Error(codes.InvalidArgument, "end_time must be in the future")
	}

	// Предупреждение если интервал слишком короткий (меньше 1 минуты)
	duration := end.Sub(start)
	if duration < time.Minute {
		s.logger.Warn("Very short lock duration",
			zap.Duration("duration", duration),
			zap.String("recommendation", "consider using longer lock period"),
		)
	}

	return nil
}

// mapRASError мапит RAS ошибки в gRPC status codes
// CRITICAL #4: Expanded error mapping with 8 error types
func (s *InfobaseManagementServer) mapRASError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := strings.ToLower(err.Error())

	// NotFound - ресурс не найден
	if strings.Contains(errMsg, "not found") || strings.Contains(errMsg, "does not exist") {
		return status.Error(codes.NotFound, "resource not found")
	}

	// PermissionDenied - нет прав доступа
	if strings.Contains(errMsg, "access denied") ||
	   strings.Contains(errMsg, "permission denied") ||
	   strings.Contains(errMsg, "unauthorized") {
		return status.Error(codes.PermissionDenied, "access denied")
	}

	// AlreadyExists - ресурс уже существует
	if strings.Contains(errMsg, "already exists") ||
	   strings.Contains(errMsg, "duplicate") {
		return status.Error(codes.AlreadyExists, "resource already exists")
	}

	// InvalidArgument - некорректные параметры
	if strings.Contains(errMsg, "invalid") ||
	   strings.Contains(errMsg, "bad request") ||
	   strings.Contains(errMsg, "malformed") {
		return status.Error(codes.InvalidArgument, "invalid request parameters")
	}

	// Unauthenticated - ошибка аутентификации
	if strings.Contains(errMsg, "authentication failed") ||
	   strings.Contains(errMsg, "invalid credentials") ||
	   strings.Contains(errMsg, "bad password") {
		return status.Error(codes.Unauthenticated, "authentication failed")
	}

	// Unavailable - сервис недоступен
	if strings.Contains(errMsg, "connection refused") ||
	   strings.Contains(errMsg, "timeout") ||
	   strings.Contains(errMsg, "unavailable") ||
	   strings.Contains(errMsg, "connection failed") {
		return status.Error(codes.Unavailable, "RAS service unavailable")
	}

	// ResourceExhausted - превышены лимиты
	if strings.Contains(errMsg, "quota exceeded") ||
	   strings.Contains(errMsg, "too many") ||
	   strings.Contains(errMsg, "limit exceeded") {
		return status.Error(codes.ResourceExhausted, "resource limit exceeded")
	}

	// FailedPrecondition - состояние не позволяет операцию
	if strings.Contains(errMsg, "locked") ||
	   strings.Contains(errMsg, "in use") ||
	   strings.Contains(errMsg, "busy") {
		return status.Error(codes.FailedPrecondition, "resource is locked or busy")
	}

	return status.Error(codes.Internal, fmt.Sprintf("RAS error: %v", err))
}

// sanitizePassword заменяет пароль на маску для логирования
func sanitizePassword(password string) string {
	if password == "" {
		return "<empty>"
	}
	return "<provided>"
}

// ==================== CRUD OPERATIONS ====================

// UpdateInfobase изменяет параметры существующей информационной базы
// Это базовый метод, который используется LockInfobase и UnlockInfobase
func (s *InfobaseManagementServer) UpdateInfobase(
	ctx context.Context,
	req *pb.UpdateInfobaseRequest,
) (*pb.UpdateInfobaseResponse, error) {
	// Валидация обязательных полей
	if err := s.validateClusterId(req.ClusterId); err != nil {
		return nil, err
	}
	if err := s.validateInfobaseId(req.InfobaseId); err != nil {
		return nil, err
	}

	// Логирование запроса (без паролей!)
	s.logger.Info("UpdateInfobase request",
		zap.String("cluster_id", req.ClusterId),
		zap.String("infobase_id", req.InfobaseId),
		zap.Any("sessions_deny", req.SessionsDeny),
		zap.Any("scheduled_jobs_deny", req.ScheduledJobsDeny),
		zap.String("db_password", sanitizePassword(req.GetDbPassword())),
		zap.String("cluster_password", sanitizePassword(req.GetClusterPassword())),
	)

	// Check context cancellation before RAS connection
	select {
	case <-ctx.Done():
		return nil, status.Errorf(codes.Canceled, "operation cancelled: %v", ctx.Err())
	default:
		// proceed
	}

	// 1. Получить endpoint от RAS client
	endpoint, err := s.client.GetEndpoint(ctx)
	if err != nil {
		s.logger.Error("Failed to get RAS endpoint",
			zap.String("cluster_id", req.ClusterId),
			zap.String("infobase_id", req.InfobaseId),
			zap.Error(err),
		)
		return nil, s.mapRASError(err)
	}

	// 2. Построить InfobaseInfo для обновления
	//    ВАЖНО: Только измененные поля! (partial update)
	infobaseInfo := &serializev1.InfobaseInfo{
		ClusterId: req.ClusterId,
		Uuid:      req.InfobaseId, // UUID для идентификации существующей базы
	}

	// Установить только переданные поля
	if req.SessionsDeny != nil {
		infobaseInfo.SessionsDeny = *req.SessionsDeny
	}
	if req.ScheduledJobsDeny != nil {
		infobaseInfo.ScheduledJobsDeny = *req.ScheduledJobsDeny
	}
	if req.DeniedMessage != nil {
		infobaseInfo.DeniedMessage = *req.DeniedMessage
	}
	if req.DeniedFrom != nil {
		infobaseInfo.DeniedFrom = req.DeniedFrom
	}
	if req.DeniedTo != nil {
		infobaseInfo.DeniedTo = req.DeniedTo
	}
	if req.PermissionCode != nil {
		infobaseInfo.PermissionCode = *req.PermissionCode
	}
	if req.Dbms != nil {
		infobaseInfo.Dbms = mapDBMSTypeToString(*req.Dbms)
	}
	if req.DbServer != nil {
		infobaseInfo.DbServer = *req.DbServer
	}
	if req.DbName != nil {
		infobaseInfo.DbName = *req.DbName
	}
	if req.DbUser != nil {
		infobaseInfo.DbUser = *req.DbUser
	}
	if req.DbPassword != nil {
		infobaseInfo.DbPwd = *req.DbPassword
	}
	if req.Description != nil {
		infobaseInfo.Descr = *req.Description
	}
	if req.SecurityLevel != nil {
		infobaseInfo.SecurityLevel = mapSecurityLevelToInt(*req.SecurityLevel)
	}

	// 3. Упаковать в EndpointRequest
	anyRequest, err := anypb.New(infobaseInfo)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to marshal request")
	}

	anyRespond, err := anypb.New(&serializev1.InfobaseInfo{})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create response template")
	}

	endpointReq := &clientv1.EndpointRequest{
		Request: anyRequest,
		Respond: anyRespond,
	}

	// Check context cancellation before expensive RAS operation
	select {
	case <-ctx.Done():
		return nil, status.Errorf(codes.Canceled, "operation cancelled before RAS request: %v", ctx.Err())
	default:
		// proceed
	}

	// 4. Выполнить запрос
	responseAny, err := endpoint.Request(ctx, endpointReq)
	if err != nil {
		s.logger.Error("Failed to update infobase via RAS",
			zap.String("cluster_id", req.ClusterId),
			zap.String("infobase_id", req.InfobaseId),
			zap.Error(err),
		)
		return nil, s.mapRASError(err)
	}

	// 5. Распаковать ответ
	var updatedInfobase serializev1.InfobaseInfo
	if err := anypb.UnmarshalTo(responseAny, &updatedInfobase, proto.UnmarshalOptions{}); err != nil {
		return nil, status.Error(codes.Internal, "failed to unmarshal response")
	}

	// 6. Success logging
	s.logger.Info("Infobase updated successfully",
		zap.String("cluster_id", req.ClusterId),
		zap.String("infobase_id", req.InfobaseId),
		zap.Any("sessions_deny", req.SessionsDeny),
		zap.Any("scheduled_jobs_deny", req.ScheduledJobsDeny),
	)

	return &pb.UpdateInfobaseResponse{
		InfobaseId: req.InfobaseId,
		Message:    "Infobase updated successfully",
		Success:    true,
	}, nil
}

// CreateInfobase создает новую информационную базу в кластере
func (s *InfobaseManagementServer) CreateInfobase(
	ctx context.Context,
	req *pb.CreateInfobaseRequest,
) (*pb.CreateInfobaseResponse, error) {
	// Валидация обязательных полей
	if err := s.validateClusterId(req.ClusterId); err != nil {
		return nil, err
	}
	if err := s.validateName(req.Name); err != nil {
		return nil, err
	}
	if err := s.validateDBMS(req.Dbms); err != nil {
		return nil, err
	}

	// CRITICAL #2: Валидация серверных полей для всех СУБД
	// Все типы в protobuf - серверные: MSSQL, PostgreSQL, IBM_DB2, Oracle
	if req.Dbms != pb.DBMSType_DBMS_TYPE_UNSPECIFIED {
		if strings.TrimSpace(req.DbServer) == "" {
			return nil, status.Error(codes.InvalidArgument, "db_server is required")
		}
		if strings.TrimSpace(req.DbName) == "" {
			return nil, status.Error(codes.InvalidArgument, "db_name is required")
		}
	}

	// Логирование запроса (без паролей!)
	s.logger.Info("CreateInfobase request",
		zap.String("cluster_id", req.ClusterId),
		zap.String("name", req.Name),
		zap.String("dbms", req.Dbms.String()),
		zap.String("db_server", req.DbServer),
		zap.String("db_name", req.DbName),
		zap.String("db_user", req.GetDbUser()),
		zap.String("db_password", sanitizePassword(req.GetDbPassword())),
		zap.String("cluster_password", sanitizePassword(req.GetClusterPassword())),
	)

	// Check context cancellation before RAS connection
	select {
	case <-ctx.Done():
		return nil, status.Errorf(codes.Canceled, "operation cancelled: %v", ctx.Err())
	default:
		// proceed
	}

	// 1. Получить endpoint от RAS client
	endpoint, err := s.client.GetEndpoint(ctx)
	if err != nil {
		s.logger.Error("Failed to get RAS endpoint",
			zap.String("cluster_id", req.ClusterId),
			zap.Error(err),
		)
		return nil, s.mapRASError(err)
	}

	// 2. IDEMPOTENCY CHECK: Проверяем существование базы с таким же именем
	existingInfobase, err := s.findInfobaseByName(ctx, endpoint, req.ClusterId, req.Name)
	if err != nil && status.Code(err) != codes.NotFound {
		// Ошибка при поиске (не NotFound) → возвращаем ошибку
		s.logger.Error("Failed to check existing infobase",
			zap.String("cluster_id", req.ClusterId),
			zap.String("name", req.Name),
			zap.Error(err),
		)
		return nil, err
	}


	if existingInfobase != nil {
		// База уже существует с таким же именем
		// Для строгой idempotency возвращаем успех с существующим UUID
		s.logger.Info("Idempotent CreateInfobase request",
			zap.String("operation", "CreateInfobase"),
			zap.String("cluster_id", req.ClusterId),
			zap.String("name", req.Name),
			zap.String("existing_infobase_id", existingInfobase.GetUuid()),
			zap.String("result", "infobase_already_exists"),
		)

		return &pb.CreateInfobaseResponse{
			InfobaseId: existingInfobase.GetUuid(),
			Name:       existingInfobase.GetName(),
			Message:    "Infobase already exists (idempotent operation)",
		}, nil
	}

	// 3. База не существует → создаем как обычно

	// 4. Построить InfobaseInfo для создания
	infobaseInfo := &serializev1.InfobaseInfo{
		ClusterId:           req.ClusterId,
		Name:                req.Name,
		Dbms:                mapDBMSTypeToString(req.Dbms),
		DbServer:            req.DbServer,
		DbName:              req.DbName,
		DbUser:              req.GetDbUser(),
		DbPwd:               req.GetDbPassword(),
		DateOffset:          req.GetDateOffset(),
		Locale:              req.GetLocale(),
		Descr:               req.GetDescription(),
		SecurityLevel:       mapSecurityLevelToInt(req.GetSecurityLevel()),
		ScheduledJobsDeny:   req.GetScheduledJobsDeny(),
		LicenseDistribution: mapLicenseDistributionToInt(req.GetLicenseDistributionAllow()),
	}

	// 5. Упаковать в Any для EndpointRequest
	anyRequest, err := anypb.New(infobaseInfo)
	if err != nil {
		s.logger.Error("Failed to marshal InfobaseInfo",
			zap.String("cluster_id", req.ClusterId),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, "failed to marshal request")
	}

	// 6. Создать шаблон ответа
	anyRespond, err := anypb.New(&serializev1.InfobaseInfo{})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create response template")
	}

	// 7. Построить EndpointRequest
	endpointReq := &clientv1.EndpointRequest{
		Request: anyRequest,
		Respond: anyRespond,
	}

	// Check context cancellation before expensive RAS operation
	select {
	case <-ctx.Done():
		return nil, status.Errorf(codes.Canceled, "operation cancelled before RAS request: %v", ctx.Err())
	default:
		// proceed
	}

	// 8. Выполнить запрос через RAS endpoint
	responseAny, err := endpoint.Request(ctx, endpointReq)
	if err != nil {
		s.logger.Error("Failed to create infobase via RAS",
			zap.String("cluster_id", req.ClusterId),
			zap.String("name", req.Name),
			zap.Error(err),
		)
		return nil, s.mapRASError(err)
	}

	// 9. Распаковать ответ
	var createdInfobase serializev1.InfobaseInfo
	if err := anypb.UnmarshalTo(responseAny, &createdInfobase, proto.UnmarshalOptions{}); err != nil {
		s.logger.Error("Failed to unmarshal RAS response",
			zap.String("cluster_id", req.ClusterId),
			zap.Error(err),
		)
		return nil, status.Error(codes.Internal, "failed to unmarshal response")
	}

	// 10. Success logging
	s.logger.Info("Infobase created successfully",
		zap.String("cluster_id", req.ClusterId),
		zap.String("infobase_id", createdInfobase.Uuid),
		zap.String("name", createdInfobase.Name),
	)

	return &pb.CreateInfobaseResponse{
		InfobaseId: createdInfobase.Uuid,
		Name:       createdInfobase.Name,
		Message:    "Infobase created successfully",
	}, nil
}

// DropInfobase удаляет информационную базу из кластера
// КРИТИЧНО: Деструктивная операция с обязательным audit logging
//
// ВАЖНО: RAS Binary Protocol поддерживает ТОЛЬКО режим DROP_MODE_UNREGISTER_ONLY.
// Режимы DROP_DATABASE и CLEAR_DATABASE не поддерживаются протоколом и вернут Unimplemented.
func (s *InfobaseManagementServer) DropInfobase(
	ctx context.Context,
	req *pb.DropInfobaseRequest,
) (*pb.DropInfobaseResponse, error) {
	// Валидация обязательных полей
	if err := s.validateClusterId(req.ClusterId); err != nil {
		return nil, err
	}
	if err := s.validateInfobaseId(req.InfobaseId); err != nil {
		return nil, err
	}

	// Валидация режима удаления
	if req.DropMode == pb.DropMode_DROP_MODE_UNSPECIFIED {
		return nil, status.Error(codes.InvalidArgument, "drop_mode is required")
	}

	// ⚠️ КРИТИЧНО: Проверка поддержки drop_mode
	// RAS Binary Protocol НЕ поддерживает DROP_DATABASE и CLEAR_DATABASE режимы
	if req.DropMode != pb.DropMode_DROP_MODE_UNREGISTER_ONLY {
		s.logger.Warn("Unsupported drop_mode requested",
			zap.String("operation", "DropInfobase"),
			zap.String("cluster_id", req.ClusterId),
			zap.String("infobase_id", req.InfobaseId),
			zap.String("drop_mode", req.DropMode.String()),
			zap.String("reason", "RAS Binary Protocol limitation"),
		)
		return nil, status.Errorf(
			codes.Unimplemented,
			"drop_mode %s is not supported by RAS Binary Protocol. Only DROP_MODE_UNREGISTER_ONLY is available. "+
			"To drop database files, use external database management tools after unregistering the infobase.",
			req.DropMode.String(),
		)
	}

	// ⚠️ AUDIT LOG ПЕРЕД операцией
	s.logger.Warn("Destructive operation requested",
		zap.String("operation", "DropInfobase"),
		zap.String("cluster_id", req.ClusterId),
		zap.String("infobase_id", req.InfobaseId),
		zap.String("drop_mode", req.DropMode.String()),
		zap.String("cluster_user", req.GetClusterUser()),
		zap.Time("requested_at", time.Now()),
	)

	// Check context cancellation before RAS connection
	select {
	case <-ctx.Done():
		// Audit log for cancelled destructive operation
		s.logger.Info("Destructive operation CANCELLED",
			zap.String("operation", "DropInfobase"),
			zap.String("cluster_id", req.ClusterId),
			zap.String("infobase_id", req.InfobaseId),
			zap.String("status", "cancelled"),
			zap.Error(ctx.Err()),
		)
		return nil, status.Errorf(codes.Canceled, "operation cancelled: %v", ctx.Err())
	default:
		// proceed
	}

	// 1. Получить endpoint от RAS client
	endpoint, err := s.client.GetEndpoint(ctx)
	if err != nil {
		s.logger.Error("Failed to get RAS endpoint",
			zap.String("cluster_id", req.ClusterId),
			zap.String("infobase_id", req.InfobaseId),
			zap.Error(err),
		)
		return nil, s.mapRASError(err)
	}

	// 2. Построить DeleteInfobaseRequest
	// ВАЖНО: RAS использует DELETE_INFOBASE_REQUEST message type
	// Нужно передать cluster_id, infobase_id, и drop_mode
	deleteRequest := &serializev1.InfobaseInfo{
		ClusterId: req.ClusterId,
		Uuid:      req.InfobaseId,
		// drop_mode кодируется через специальное поле или отдельный message
		// Здесь используем InfobaseInfo с UUID для идентификации
	}

	// 3. Упаковать в EndpointRequest
	anyRequest, err := anypb.New(deleteRequest)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to marshal request")
	}

	// Response - используем InfobaseInfo для получения подтверждения
	anyRespond, err := anypb.New(&serializev1.InfobaseInfo{})
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to create response template")
	}

	endpointReq := &clientv1.EndpointRequest{
		Request: anyRequest,
		Respond: anyRespond,
	}

	// Check context cancellation before expensive RAS operation
	select {
	case <-ctx.Done():
		// Audit log for cancelled destructive operation
		s.logger.Info("Destructive operation CANCELLED before RAS request",
			zap.String("operation", "DropInfobase"),
			zap.String("cluster_id", req.ClusterId),
			zap.String("infobase_id", req.InfobaseId),
			zap.String("status", "cancelled"),
			zap.Error(ctx.Err()),
		)
		return nil, status.Errorf(codes.Canceled, "operation cancelled before RAS request: %v", ctx.Err())
	default:
		// proceed
	}

	// 4. Выполнить запрос
	_, dropErr := endpoint.Request(ctx, endpointReq)

	// 5. AUDIT LOG ПОСЛЕ операции
	if dropErr != nil {
		s.logger.Error("Destructive operation FAILED",
			zap.String("operation", "DropInfobase"),
			zap.String("infobase_id", req.InfobaseId),
			zap.String("drop_mode", req.DropMode.String()),
			zap.String("cluster_id", req.ClusterId),
			zap.Error(dropErr),
			zap.Time("failed_at", time.Now()),
		)
		return nil, s.mapRASError(dropErr)
	}

	// SUCCESS audit log
	s.logger.Warn("Destructive operation COMPLETED",
		zap.String("operation", "DropInfobase"),
		zap.String("infobase_id", req.InfobaseId),
		zap.String("drop_mode", req.DropMode.String()),
		zap.String("cluster_id", req.ClusterId),
		zap.Time("completed_at", time.Now()),
		zap.Bool("success", true),
	)

	return &pb.DropInfobaseResponse{
		InfobaseId: req.InfobaseId,
		Message:    "Infobase dropped successfully",
		Success:    true,
	}, nil
}

// ==================== LOCK/UNLOCK OPERATIONS ====================
// Эти методы являются wrapper'ами над UpdateInfobase

// LockInfobase блокирует доступ к информационной базе
// Это wrapper над UpdateInfobase с установкой флагов блокировки
func (s *InfobaseManagementServer) LockInfobase(
	ctx context.Context,
	req *pb.LockInfobaseRequest,
) (*pb.LockInfobaseResponse, error) {
	// Валидация обязательных полей
	if err := s.validateClusterId(req.ClusterId); err != nil {
		return nil, err
	}
	if err := s.validateInfobaseId(req.InfobaseId); err != nil {
		return nil, err
	}

	// CRITICAL #3: Validate lock schedule if provided
	if err := s.validateLockSchedule(req.DeniedFrom, req.DeniedTo); err != nil {
		return nil, err
	}

	s.logger.Info("LockInfobase request",
		zap.String("cluster_id", req.ClusterId),
		zap.String("infobase_id", req.InfobaseId),
		zap.Bool("sessions_deny", req.SessionsDeny),
		zap.Bool("scheduled_jobs_deny", req.ScheduledJobsDeny),
		zap.Bool("has_permission_code", req.PermissionCode != nil),
	)

	// Check context cancellation before calling UpdateInfobase
	select {
	case <-ctx.Done():
		return nil, status.Errorf(codes.Canceled, "operation cancelled: %v", ctx.Err())
	default:
		// proceed
	}

	// Построить UpdateInfobaseRequest с параметрами блокировки
	updateReq := &pb.UpdateInfobaseRequest{
		ClusterId:       req.ClusterId,
		InfobaseId:      req.InfobaseId,
		ClusterUser:     req.ClusterUser,
		ClusterPassword: req.ClusterPassword,
	}

	// Установить флаги блокировки
	if req.SessionsDeny {
		updateReq.SessionsDeny = &req.SessionsDeny
		updateReq.DeniedFrom = req.DeniedFrom
		updateReq.DeniedTo = req.DeniedTo
		updateReq.DeniedMessage = req.DeniedMessage

		// Установить permission_code если передан
		if req.PermissionCode != nil {
			updateReq.PermissionCode = req.PermissionCode
		}
	}

	if req.ScheduledJobsDeny {
		updateReq.ScheduledJobsDeny = &req.ScheduledJobsDeny
	}

	// Вызвать UpdateInfobase
	updateResp, err := s.UpdateInfobase(ctx, updateReq)
	if err != nil {
		return nil, err
	}

	// Вернуть ответ с текущим временем
	return &pb.LockInfobaseResponse{
		InfobaseId: req.InfobaseId,
		Message:    updateResp.Message,
		Success:    updateResp.Success,
	}, nil
}

// UnlockInfobase снимает блокировку с информационной базы
// Это wrapper над UpdateInfobase с снятием флагов блокировки
func (s *InfobaseManagementServer) UnlockInfobase(
	ctx context.Context,
	req *pb.UnlockInfobaseRequest,
) (*pb.UnlockInfobaseResponse, error) {
	// Валидация обязательных полей
	if err := s.validateClusterId(req.ClusterId); err != nil {
		return nil, err
	}
	if err := s.validateInfobaseId(req.InfobaseId); err != nil {
		return nil, err
	}

	s.logger.Info("UnlockInfobase request",
		zap.String("cluster_id", req.ClusterId),
		zap.String("infobase_id", req.InfobaseId),
		zap.Bool("unlock_sessions", req.UnlockSessions),
		zap.Bool("unlock_scheduled_jobs", req.UnlockScheduledJobs),
	)

	// Check context cancellation before calling UpdateInfobase
	select {
	case <-ctx.Done():
		return nil, status.Errorf(codes.Canceled, "operation cancelled: %v", ctx.Err())
	default:
		// proceed
	}

	// Построить UpdateInfobaseRequest с параметрами разблокировки
	updateReq := &pb.UpdateInfobaseRequest{
		ClusterId:       req.ClusterId,
		InfobaseId:      req.InfobaseId,
		ClusterUser:     req.ClusterUser,
		ClusterPassword: req.ClusterPassword,
	}

	// Снять флаги блокировки
	if req.UnlockSessions {
		sessionsDeny := false
		updateReq.SessionsDeny = &sessionsDeny
		// Очистить permission_code
		emptyCode := ""
		updateReq.PermissionCode = &emptyCode
	}

	if req.UnlockScheduledJobs {
		scheduledJobsDeny := false
		updateReq.ScheduledJobsDeny = &scheduledJobsDeny
	}

	// Вызвать UpdateInfobase
	updateResp, err := s.UpdateInfobase(ctx, updateReq)
	if err != nil {
		return nil, err
	}

	// Вернуть ответ с текущим временем
	return &pb.UnlockInfobaseResponse{
		InfobaseId: req.InfobaseId,
		Message:    updateResp.Message,
		Success:    updateResp.Success,
	}, nil
}

// ==================== HELPER FUNCTIONS ====================

// mapDBMSTypeToString converts protobuf enum to string for RAS
func mapDBMSTypeToString(dbms pb.DBMSType) string {
	switch dbms {
	case pb.DBMSType_DBMS_TYPE_MSSQL_SERVER:
		return "MSSQLServer"
	case pb.DBMSType_DBMS_TYPE_POSTGRESQL:
		return "PostgreSQL"
	case pb.DBMSType_DBMS_TYPE_IBM_DB2:
		return "IBMDB2"
	case pb.DBMSType_DBMS_TYPE_ORACLE:
		return "OracleDatabase"
	default:
		return ""
	}
}

// mapSecurityLevelToInt converts SecurityLevel enum to int32 for RAS
func mapSecurityLevelToInt(level pb.SecurityLevel) int32 {
	switch level {
	case pb.SecurityLevel_SECURITY_LEVEL_0:
		return 0
	case pb.SecurityLevel_SECURITY_LEVEL_1:
		return 1
	case pb.SecurityLevel_SECURITY_LEVEL_2:
		return 2
	case pb.SecurityLevel_SECURITY_LEVEL_3:
		return 3
	default:
		return 0
	}
}

// mapLicenseDistributionToInt converts bool to int32 for RAS
// allow=true -> 0 (разрешено), allow=false -> 1 (запрещено)
func mapLicenseDistributionToInt(allow bool) int32 {
	if allow {
		return 0 // разрешено
	}
	return 1 // запрещено
}
