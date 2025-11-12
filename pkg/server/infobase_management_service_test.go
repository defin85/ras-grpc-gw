package server

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	pb "github.com/v8platform/ras-grpc-gw/pkg/gen/infobase/service"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func createTestLogger() (*zap.Logger, *observer.ObservedLogs) {
	core, logs := observer.New(zapcore.InfoLevel)
	return zap.New(core), logs
}

func TestValidateClusterId(t *testing.T) {
	srv := &InfobaseManagementServer{logger: zap.NewNop()}
	tests := []struct {
		name      string
		clusterId string
		wantErr   bool
	}{
		{"valid UUID", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid name", "cluster1", false},
		{"empty", "", true},
		{"whitespace", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := srv.validateClusterId(tt.clusterId)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateInfobaseId(t *testing.T) {
	srv := &InfobaseManagementServer{logger: zap.NewNop()}
	tests := []struct {
		name       string
		infobaseId string
		wantErr    bool
	}{
		{"valid UUID", "660e8400-e29b-41d4-a716-446655440000", false},
		{"valid name", "ib1", false},
		{"empty", "", true},
		{"whitespace", "   ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := srv.validateInfobaseId(tt.infobaseId)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	srv := &InfobaseManagementServer{logger: zap.NewNop()}
	tests := []struct {
		name      string
		inputName string
		wantErr   bool
	}{
		// Latin names
		{"valid alphanumeric", "TestInfobase123", false},
		{"valid with hyphen", "test-infobase", false},
		{"valid with underscore", "test_infobase_db", false},
		{"valid 64 chars", "a123456789012345678901234567890123456789012345678901234567890123", false},

		// Cyrillic names - NOW VALID
		{"cyrillic valid", "Бухгалтерия_2024", false},
		{"cyrillic uppercase valid", "БУХГАЛТЕРИЯ_ТОП", false},
		{"cyrillic lowercase valid", "тестовая_база", false},
		{"mixed latin and cyrillic valid", "База_1C_Торговля", false},
		{"cyrillic with hyphen valid", "Моя-База-Данных", false},
		{"cyrillic with number", "УТ11_Основная", false},
		{"cyrillic with ё", "Учёт_Склад", false},

		// Other Unicode letters
		{"chinese valid", "会计_2024", false},
		{"german valid", "Datenbank_Übung", false},
		{"french valid", "Base-de-Données", false},

		// Invalid cases
		{"empty", "", true},
		{"whitespace", "   ", true},
		{"with spaces", "Test Infobase", true},
		{"with @", "test@base", true},
		{"with special char #", "test#base", true},
		{"with special char $", "test$base", true},
		{"with emoji", "Test🔒Base", true},
		{"cyrillic with space", "База Данных", true},
		{"cyrillic with @", "База@2024", true},
		{"65 chars", "a1234567890123456789012345678901234567890123456789012345678901234", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := srv.validateName(tt.inputName)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateDBMS(t *testing.T) {
	srv := &InfobaseManagementServer{logger: zap.NewNop()}
	tests := []struct {
		name     string
		dbmsType pb.DBMSType
		wantErr  bool
	}{
		{"PostgreSQL", pb.DBMSType_DBMS_TYPE_POSTGRESQL, false},
		{"MSSQL", pb.DBMSType_DBMS_TYPE_MSSQL_SERVER, false},
		{"DB2", pb.DBMSType_DBMS_TYPE_IBM_DB2, false},
		{"Oracle", pb.DBMSType_DBMS_TYPE_ORACLE, false},
		{"Unspecified", pb.DBMSType_DBMS_TYPE_UNSPECIFIED, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := srv.validateDBMS(tt.dbmsType)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateLockSchedule(t *testing.T) {
	logger, _ := createTestLogger()
	srv := &InfobaseManagementServer{logger: logger}
	now := time.Now()
	future1h := now.Add(1 * time.Hour)
	future2h := now.Add(2 * time.Hour)
	past1h := now.Add(-1 * time.Hour)

	tests := []struct {
		name      string
		startTime *timestamppb.Timestamp
		endTime   *timestamppb.Timestamp
		wantErr   bool
	}{
		{"permanent lock", nil, nil, false},
		{"valid schedule", timestamppb.New(future1h), timestamppb.New(future2h), false},
		{"only start", timestamppb.New(future1h), nil, true},
		{"only end", nil, timestamppb.New(future2h), true},
		{"end before start", timestamppb.New(future2h), timestamppb.New(future1h), true},
		{"end in past", timestamppb.New(past1h), timestamppb.New(now.Add(-30 * time.Minute)), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := srv.validateLockSchedule(tt.startTime, tt.endTime)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMapDBMSTypeToString(t *testing.T) {
	tests := []struct {
		dbmsType pb.DBMSType
		want     string
	}{
		{pb.DBMSType_DBMS_TYPE_POSTGRESQL, "PostgreSQL"},
		{pb.DBMSType_DBMS_TYPE_MSSQL_SERVER, "MSSQLServer"},
		{pb.DBMSType_DBMS_TYPE_IBM_DB2, "IBMDB2"},
		{pb.DBMSType_DBMS_TYPE_ORACLE, "OracleDatabase"},
		{pb.DBMSType_DBMS_TYPE_UNSPECIFIED, ""},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := mapDBMSTypeToString(tt.dbmsType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMapSecurityLevelToInt(t *testing.T) {
	tests := []struct {
		level pb.SecurityLevel
		want  int32
	}{
		{pb.SecurityLevel_SECURITY_LEVEL_0, 0},
		{pb.SecurityLevel_SECURITY_LEVEL_1, 1},
		{pb.SecurityLevel_SECURITY_LEVEL_2, 2},
		{pb.SecurityLevel_SECURITY_LEVEL_3, 3},
		{pb.SecurityLevel_SECURITY_LEVEL_UNSPECIFIED, 0},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := mapSecurityLevelToInt(tt.level)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMapLicenseDistributionToInt(t *testing.T) {
	assert.Equal(t, int32(0), mapLicenseDistributionToInt(true))
	assert.Equal(t, int32(1), mapLicenseDistributionToInt(false))
}

func TestMapRASError(t *testing.T) {
	srv := &InfobaseManagementServer{logger: zap.NewNop()}
	tests := []struct {
		name     string
		rasError error
		wantCode codes.Code
	}{
		{"nil", nil, codes.OK},
		{"not found", errors.New("not found"), codes.NotFound},
		{"access denied", errors.New("access denied"), codes.PermissionDenied},
		{"already exists", errors.New("already exists"), codes.AlreadyExists},
		{"invalid", errors.New("invalid parameter"), codes.InvalidArgument},
		{"authentication failed", errors.New("authentication failed"), codes.Unauthenticated},
		{"timeout", errors.New("timeout"), codes.Unavailable},
		{"quota exceeded", errors.New("quota exceeded"), codes.ResourceExhausted},
		{"locked", errors.New("database locked"), codes.FailedPrecondition},
		{"unknown", errors.New("something went wrong"), codes.Internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := srv.mapRASError(tt.rasError)
			if tt.rasError == nil {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			st, ok := status.FromError(err)
			require.True(t, ok)
			assert.Equal(t, tt.wantCode, st.Code())
		})
	}
}

func TestSanitizePassword(t *testing.T) {
	assert.Equal(t, "<empty>", sanitizePassword(""))
	assert.Equal(t, "<provided>", sanitizePassword("secret123"))
	assert.Equal(t, "<provided>", sanitizePassword("a"))
}

func TestSanitizePassword_NoLeak(t *testing.T) {
	logger, logs := createTestLogger()
	realPassword := "SuperSecret123"
	logger.Info("Test", zap.String("pwd", sanitizePassword(realPassword)))

	allLogs := logs.All()
	require.Len(t, allLogs, 1)
	
	for _, field := range allLogs[0].Context {
		if field.Key == "pwd" {
			assert.Equal(t, "<provided>", field.String)
			assert.NotContains(t, field.String, "SuperSecret")
		}
	}
}
