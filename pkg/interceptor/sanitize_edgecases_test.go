package interceptor

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// TestSanitizePasswordsInterceptor_MultiplePasswords tests sanitization of multiple password fields in one request
func TestSanitizePasswordsInterceptor_MultiplePasswords(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := SanitizePasswordsInterceptor(logger)

	// Create request with 3 different password fields
	req := createTestMessage("cluster_secret", "db_secret", "infobase_secret")

	ctx := context.Background()
	info := mockServerInfo("/test.Service/MultiPasswordMethod")

	resp, err := interceptor(ctx, req, info, mockHandler)

	require.NoError(t, err)
	assert.Equal(t, "response", resp)

	// Verify original request is NOT modified
	reflectMsg := req.ProtoReflect()
	msgType := reflectMsg.Type().Descriptor()

	clusterPwd := reflectMsg.Get(msgType.Fields().ByName("cluster_password")).String()
	dbPwd := reflectMsg.Get(msgType.Fields().ByName("db_password")).String()
	infobasePwd := reflectMsg.Get(msgType.Fields().ByName("infobase_password")).String()

	// All original passwords should remain unchanged
	assert.Equal(t, "cluster_secret", clusterPwd)
	assert.Equal(t, "db_secret", dbPwd)
	assert.Equal(t, "infobase_secret", infobasePwd)
}

// TestSanitizePasswordsInterceptor_UnicodePasswords tests sanitization of passwords with unicode characters
func TestSanitizePasswordsInterceptor_UnicodePasswords(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := SanitizePasswordsInterceptor(logger)

	tests := []struct {
		name     string
		password string
	}{
		{"cyrillic", "пароль123"},
		{"emoji", "pass🔒word"},
		{"chinese", "密码123"},
		{"mixed", "пас🔐word密码"},
		{"special chars", "p@ss!#$%^&*()"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createTestMessage(tt.password, "", "")

			ctx := context.Background()
			info := mockServerInfo("/test.Service/Method")

			resp, err := interceptor(ctx, req, info, mockHandler)

			require.NoError(t, err)
			assert.Equal(t, "response", resp)

			// Original should remain unchanged
			reflectMsg := req.ProtoReflect()
			msgType := reflectMsg.Type().Descriptor()
			clusterPwd := reflectMsg.Get(msgType.Fields().ByName("cluster_password")).String()
			assert.Equal(t, tt.password, clusterPwd)
		})
	}
}

// TestSanitizePasswordsInterceptor_VeryLongPassword tests sanitization of very long passwords
func TestSanitizePasswordsInterceptor_VeryLongPassword(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := SanitizePasswordsInterceptor(logger)

	// Create 1000-character password
	longPassword := string(make([]byte, 1000))
	for i := range longPassword {
		longPassword = longPassword[:i] + "a"
	}
	longPassword = ""
	for i := 0; i < 1000; i++ {
		longPassword += "a"
	}

	req := createTestMessage(longPassword, "", "")

	ctx := context.Background()
	info := mockServerInfo("/test.Service/Method")

	resp, err := interceptor(ctx, req, info, mockHandler)

	require.NoError(t, err)
	assert.Equal(t, "response", resp)

	// Original should remain unchanged
	reflectMsg := req.ProtoReflect()
	msgType := reflectMsg.Type().Descriptor()
	clusterPwd := reflectMsg.Get(msgType.Fields().ByName("cluster_password")).String()
	assert.Equal(t, longPassword, clusterPwd)
	assert.Equal(t, 1000, len(clusterPwd))
}

// TestSanitizePasswordsInterceptor_Concurrent tests concurrent calls to interceptor
func TestSanitizePasswordsInterceptor_Concurrent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := SanitizePasswordsInterceptor(logger)

	const numGoroutines = 100
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(index int) {
			defer wg.Done()

			// Each goroutine has unique password
			password := "secret" + string(rune(index))
			req := createTestMessage(password, "", "")

			ctx := context.Background()
			info := mockServerInfo("/test.Service/Method")

			resp, err := interceptor(ctx, req, info, mockHandler)
			if err != nil {
				errors <- err
				return
			}

			// Verify response
			if resp != "response" {
				errors <- assert.AnError
				return
			}

			// Verify original password unchanged
			reflectMsg := req.ProtoReflect()
			msgType := reflectMsg.Type().Descriptor()
			clusterPwd := reflectMsg.Get(msgType.Fields().ByName("cluster_password")).String()
			if clusterPwd != password {
				errors <- assert.AnError
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		if err != nil {
			errorCount++
			t.Errorf("Concurrent test error: %v", err)
		}
	}

	assert.Equal(t, 0, errorCount, "Should have no errors in concurrent execution")
}

// TestSanitizePasswordsInterceptor_MixedEmptyAndFilled tests mix of empty and filled passwords
func TestSanitizePasswordsInterceptor_MixedEmptyAndFilled(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := SanitizePasswordsInterceptor(logger)

	tests := []struct {
		name              string
		clusterPassword   string
		dbPassword        string
		infobasePassword  string
	}{
		{"first empty", "", "secret2", "secret3"},
		{"middle empty", "secret1", "", "secret3"},
		{"last empty", "secret1", "secret2", ""},
		{"first and last empty", "", "secret2", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createTestMessage(tt.clusterPassword, tt.dbPassword, tt.infobasePassword)

			ctx := context.Background()
			info := mockServerInfo("/test.Service/Method")

			resp, err := interceptor(ctx, req, info, mockHandler)

			require.NoError(t, err)
			assert.Equal(t, "response", resp)

			// Verify originals unchanged
			reflectMsg := req.ProtoReflect()
			msgType := reflectMsg.Type().Descriptor()

			assert.Equal(t, tt.clusterPassword, reflectMsg.Get(msgType.Fields().ByName("cluster_password")).String())
			assert.Equal(t, tt.dbPassword, reflectMsg.Get(msgType.Fields().ByName("db_password")).String())
			assert.Equal(t, tt.infobasePassword, reflectMsg.Get(msgType.Fields().ByName("infobase_password")).String())
		})
	}
}

// TestSanitizeMessage_NestedMessages tests sanitization doesn't break on nested messages (even if we don't recurse)
func TestSanitizeMessage_NestedMessages(t *testing.T) {
	// Create a simple message with password
	msg := createTestMessage("secret123", "", "")

	// Sanitize it
	sanitized := sanitizeMessage(msg)

	// Original should be unchanged
	reflectOriginal := msg.ProtoReflect()
	msgType := reflectOriginal.Type().Descriptor()
	originalPwd := reflectOriginal.Get(msgType.Fields().ByName("cluster_password")).String()
	assert.Equal(t, "secret123", originalPwd)

	// Sanitized should have masked password (in logs, but we clone so original is preserved)
	reflectSanitized := sanitized.ProtoReflect()
	sanitizedType := reflectSanitized.Type().Descriptor()
	sanitizedPwd := reflectSanitized.Get(sanitizedType.Fields().ByName("cluster_password")).String()
	assert.Equal(t, "******", sanitizedPwd)
}

// TestSanitizePasswordInString_EdgeCases tests edge cases for string sanitization helper
func TestSanitizePasswordInString_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"single char", "a", "******"},
		{"whitespace", "   ", "******"},
		{"newline", "\n", "******"},
		{"tab", "\t", "******"},
		{"unicode", "пароль", "******"},
		{"emoji", "🔒", "******"},
		{"very long", string(make([]byte, 10000)), "******"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Special case for very long string
			if tt.name == "very long" {
				longStr := ""
				for i := 0; i < 10000; i++ {
					longStr += "a"
				}
				tt.input = longStr
			}

			result := SanitizePasswordInString(tt.input)
			assert.Equal(t, tt.expected, result)

			// Verify mask is always "******" (not scaled to input length)
			if tt.input != "" {
				assert.Equal(t, "******", result)
				assert.NotEqual(t, len(tt.input), len(result), "Mask should not scale with input length")
			}
		})
	}
}

// TestSanitizePasswordsInterceptor_NilContext tests behavior with nil context (should not panic)
func TestSanitizePasswordsInterceptor_NilContext(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := SanitizePasswordsInterceptor(logger)

	req := createTestMessage("secret", "", "")
	info := mockServerInfo("/test.Service/Method")

	// This should not panic even with nil context
	resp, err := interceptor(nil, req, info, mockHandler)

	require.NoError(t, err)
	assert.Equal(t, "response", resp)
}

// Helper to create a message with only non-password fields (to test no interference)
func createTestMessageNoPasswords(name string, id int32) proto.Message {
	msgDesc := &descriptorpb.DescriptorProto{
		Name: proto.String("TestMessageNoPasswords"),
		Field: []*descriptorpb.FieldDescriptorProto{
			{
				Name:   proto.String("name"),
				Number: proto.Int32(1),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_STRING.Enum(),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			},
			{
				Name:   proto.String("id"),
				Number: proto.Int32(2),
				Type:   descriptorpb.FieldDescriptorProto_TYPE_INT32.Enum(),
				Label:  descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum(),
			},
		},
	}

	fileDesc := &descriptorpb.FileDescriptorProto{
		Name:        proto.String("test.proto"),
		Package:     proto.String("test"),
		MessageType: []*descriptorpb.DescriptorProto{msgDesc},
	}

	file, err := protodesc.NewFile(fileDesc, nil)
	if err != nil {
		panic(err)
	}

	msgType := file.Messages().Get(0)
	msg := dynamicpb.NewMessage(msgType)

	msg.Set(msgType.Fields().ByName("name"), protoreflect.ValueOfString(name))
	msg.Set(msgType.Fields().ByName("id"), protoreflect.ValueOfInt32(id))

	return msg
}

// TestSanitizePasswordsInterceptor_NoPasswordFields tests that messages without password fields work fine
func TestSanitizePasswordsInterceptor_NoPasswordFields(t *testing.T) {
	logger := zaptest.NewLogger(t)
	interceptor := SanitizePasswordsInterceptor(logger)

	req := createTestMessageNoPasswords("test-name", 123)

	ctx := context.Background()
	info := mockServerInfo("/test.Service/Method")

	resp, err := interceptor(ctx, req, info, mockHandler)

	require.NoError(t, err)
	assert.Equal(t, "response", resp)

	// Verify original unchanged
	reflectMsg := req.ProtoReflect()
	msgType := reflectMsg.Type().Descriptor()

	name := reflectMsg.Get(msgType.Fields().ByName("name")).String()
	id := reflectMsg.Get(msgType.Fields().ByName("id")).Int()

	assert.Equal(t, "test-name", name)
	assert.Equal(t, int64(123), id)
}
