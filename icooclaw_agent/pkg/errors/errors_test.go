package errors

import (
	"errors"
	"testing"
)

func TestFailoverError_Error(t *testing.T) {
	t.Run("Error with wrapped", func(t *testing.T) {
		wrappedErr := errors.New("underlying error")
		fe := &FailoverError{
			Reason:   FailoverAuth,
			Provider: "openai",
			Model:    "gpt-4",
			Status:   401,
			Wrapped:  wrappedErr,
		}

		expected := "故障转移 [auth]: underlying error (提供商=openai, 模型=gpt-4, 状态=401)"
		if fe.Error() != expected {
			t.Errorf("Error() = %q, want %q", fe.Error(), expected)
		}
	})

	t.Run("Error without wrapped", func(t *testing.T) {
		fe := &FailoverError{
			Reason:   FailoverTimeout,
			Provider: "anthropic",
			Model:    "claude-3",
			Status:   408,
			Wrapped:  nil,
		}

		expected := "故障转移 [timeout]: 提供商=anthropic, 模型=claude-3, 状态=408"
		if fe.Error() != expected {
			t.Errorf("Error() = %q, want %q", fe.Error(), expected)
		}
	})
}

func TestFailoverError_Unwrap(t *testing.T) {
	wrappedErr := errors.New("underlying")
	fe := &FailoverError{Wrapped: wrappedErr}

	if fe.Unwrap() != wrappedErr {
		t.Error("Unwrap should return wrapped error")
	}
}

func TestFailoverError_IsRetriable(t *testing.T) {
	tests := []struct {
		reason    FailoverReason
		retriable bool
	}{
		{FailoverAuth, true},
		{FailoverRateLimit, true},
		{FailoverTimeout, true},
		{FailoverFormat, false},
		{FailoverUnknown, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.reason), func(t *testing.T) {
			fe := &FailoverError{Reason: tt.reason}
			if fe.IsRetriable() != tt.retriable {
				t.Errorf("IsRetriable() = %v, want %v", fe.IsRetriable(), tt.retriable)
			}
		})
	}
}

func TestNewFailoverError(t *testing.T) {
	wrapped := errors.New("wrapped")
	fe := NewFailoverError(FailoverRateLimit, "provider", "model", 429, wrapped)

	if fe.Reason != FailoverRateLimit {
		t.Errorf("Reason = %v, want %v", fe.Reason, FailoverRateLimit)
	}
	if fe.Provider != "provider" {
		t.Errorf("Provider = %q, want %q", fe.Provider, "provider")
	}
	if fe.Model != "model" {
		t.Errorf("Model = %q, want %q", fe.Model, "model")
	}
	if fe.Status != 429 {
		t.Errorf("Status = %d, want 429", fe.Status)
	}
	if fe.Wrapped != wrapped {
		t.Error("Wrapped should be the wrapped error")
	}
}

func TestClassifiedError_Error(t *testing.T) {
	ce := &ClassifiedError{
		Code:      "ERR_TIMEOUT",
		Message:   "Request timed out",
		Retriable: true,
		Cause:     errors.New("context deadline exceeded"),
	}

	if ce.Error() != "Request timed out" {
		t.Errorf("Error() = %q, want %q", ce.Error(), "Request timed out")
	}
}

func TestClassifiedError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	ce := &ClassifiedError{Cause: cause}

	if ce.Unwrap() != cause {
		t.Error("Unwrap should return cause")
	}
}

func TestNewClassifiedError(t *testing.T) {
	cause := errors.New("cause error")
	ce := NewClassifiedError("ERR_CUSTOM", "custom error message", true, cause)

	if ce.Code != "ERR_CUSTOM" {
		t.Errorf("Code = %q, want %q", ce.Code, "ERR_CUSTOM")
	}
	if ce.Message != "custom error message" {
		t.Errorf("Message = %q, want %q", ce.Message, "custom error message")
	}
	if ce.Retriable != true {
		t.Error("Retriable should be true")
	}
	if ce.Cause != cause {
		t.Error("Cause should be the cause error")
	}
}

func TestWrap(t *testing.T) {
	original := errors.New("original error")
	wrapped := Wrap(original, "context message")

	if wrapped.Error() != "context message: original error" {
		t.Errorf("Error() = %q, want %q", wrapped.Error(), "context message: original error")
	}

	if !errors.Is(wrapped, original) {
		t.Error("errors.Is(wrapped, original) should be true")
	}
}

func TestWrapf(t *testing.T) {
	original := errors.New("original")
	wrapped := Wrapf(original, "operation %s failed", "read")

	if wrapped.Error() != "operation read failed: original" {
		t.Errorf("Error() = %q, want %q", wrapped.Error(), "operation read failed: original")
	}

	if !errors.Is(wrapped, original) {
		t.Error("errors.Is(wrapped, original) should be true")
	}
}
