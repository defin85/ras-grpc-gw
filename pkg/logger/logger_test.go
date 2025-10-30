package logger

import (
	"testing"

	"go.uber.org/zap"
)

func TestInit(t *testing.T) {
	tests := []struct {
		name    string
		debug   bool
		wantErr bool
	}{
		{
			name:    "production mode",
			debug:   false,
			wantErr: false,
		},
		{
			name:    "debug mode",
			debug:   true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Init(tt.debug)
			if (err != nil) != tt.wantErr {
				t.Errorf("Init() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err == nil && Log == nil {
				t.Error("Init() succeeded but Log is nil")
			}

			// Cleanup
			if Log != nil {
				_ = Log.Sync()
				Log = nil
			}
		})
	}
}

func TestSync(t *testing.T) {
	// Test with nil logger
	Sync() // Should not panic

	// Test with initialized logger
	if err := Init(false); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	Sync() // Should not panic

	// Cleanup
	Log = nil
}

func TestLoggerAfterInit(t *testing.T) {
	if err := Init(true); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() {
		Sync()
		Log = nil
	}()

	if Log == nil {
		t.Fatal("Log is nil after Init")
	}

	// Test that we can log without panic
	Log.Info("test message")
	Log.Debug("debug message")
	Log.Error("error message")
}

func TestLoggerProductionFormat(t *testing.T) {
	if err := Init(false); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() {
		Sync()
		Log = nil
	}()

	// Проверяем что logger работает в production режиме
	Log.Info("production test message",
		zap.String("key1", "value1"),
		zap.Int("key2", 42),
	)

	// Проверяем что можно логировать ошибки
	Log.Error("error in production", zap.Int("error_code", 500))
}

func TestLoggerDebugFormat(t *testing.T) {
	if err := Init(true); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}
	defer func() {
		Sync()
		Log = nil
	}()

	// Проверяем что logger работает в debug режиме
	Log.Debug("debug test message",
		zap.String("debug_key", "debug_value"),
	)

	Log.Warn("warning in debug mode")
}

func TestMultipleInit(t *testing.T) {
	// Первая инициализация
	if err := Init(false); err != nil {
		t.Fatalf("First Init() failed: %v", err)
	}

	firstLogger := Log

	// Вторая инициализация должна заменить логгер
	if err := Init(true); err != nil {
		t.Fatalf("Second Init() failed: %v", err)
	}

	if Log == nil {
		t.Fatal("Log is nil after second Init()")
	}

	if Log == firstLogger {
		t.Error("Second Init() did not replace the logger")
	}

	// Cleanup
	Sync()
	Log = nil
}

func TestSyncMultipleTimes(t *testing.T) {
	if err := Init(false); err != nil {
		t.Fatalf("Failed to initialize logger: %v", err)
	}

	// Вызываем Sync несколько раз - не должно быть паники
	Sync()
	Sync()
	Sync()

	// Cleanup
	Log = nil
}
