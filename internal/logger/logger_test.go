package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestInit(t *testing.T) {
	// Init should set up logger with stderr
	Init()

	// Get the default logger and verify it's not nil
	logger := slog.Default()
	if logger == nil {
		t.Fatal("default logger is nil after Init()")
	}

	// Create a test to verify logging works
	buf := &bytes.Buffer{}
	InitWithWriter(buf)
	slog.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("log output doesn't contain test message, got: %s", output)
	}
}

func TestInitWithWriter(t *testing.T) {
	tests := []struct {
		name       string
		writerFunc func() *bytes.Buffer
	}{
		{
			name: "with bytes buffer",
			writerFunc: func() *bytes.Buffer {
				return &bytes.Buffer{}
			},
		},
		{
			name: "with multiple writes",
			writerFunc: func() *bytes.Buffer {
				return &bytes.Buffer{}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := tt.writerFunc()
			InitWithWriter(buf)

			// Verify logger was set
			logger := slog.Default()
			if logger == nil {
				t.Fatal("default logger is nil after InitWithWriter()")
			}

			// Log a message and verify it's written to the buffer
			testMsg := "test with writer"
			slog.Info(testMsg)

			output := buf.String()
			if !strings.Contains(output, testMsg) {
				t.Errorf("log output doesn't contain test message, got: %s", output)
			}
		})
	}
}

func TestInitWithLevel(t *testing.T) {
	tests := []struct {
		name      string
		level     slog.Level
		testMsg   string
		shouldLog bool
	}{
		{
			name:      "with debug level",
			level:     slog.LevelDebug,
			testMsg:   "debug message",
			shouldLog: true,
		},
		{
			name:      "with info level",
			level:     slog.LevelInfo,
			testMsg:   "info message",
			shouldLog: true,
		},
		{
			name:      "with warn level",
			level:     slog.LevelWarn,
			testMsg:   "warn message",
			shouldLog: true,
		},
		{
			name:      "with error level",
			level:     slog.LevelError,
			testMsg:   "error message",
			shouldLog: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			InitWithLevel(buf, tt.level)

			logger := slog.Default()
			if logger == nil {
				t.Fatal("default logger is nil after InitWithLevel()")
			}

			// Log a message at the configured level
			switch tt.level {
			case slog.LevelDebug:
				slog.Debug(tt.testMsg)
			case slog.LevelInfo:
				slog.Info(tt.testMsg)
			case slog.LevelWarn:
				slog.Warn(tt.testMsg)
			case slog.LevelError:
				slog.Error(tt.testMsg)
			}

			output := buf.String()
			if tt.shouldLog && !strings.Contains(output, tt.testMsg) {
				t.Errorf("log output doesn't contain test message, got: %s", output)
			}
		})
	}
}

func TestInitWithLevelFiltering(t *testing.T) {
	// Test that log levels are properly filtered
	buf := &bytes.Buffer{}
	InitWithLevel(buf, slog.LevelWarn)

	logger := slog.Default()
	if logger == nil {
		t.Fatal("default logger is nil after InitWithLevel()")
	}

	// Info level should not be logged when level is Warn
	slog.Info("this should not appear")
	slog.Warn("this should appear")

	output := buf.String()
	if strings.Contains(output, "this should not appear") {
		t.Errorf("info message was logged when level is Warn")
	}
	if !strings.Contains(output, "this should appear") {
		t.Errorf("warn message was not logged, got: %s", output)
	}
}

func TestTextHandlerConfiguration(t *testing.T) {
	// Test that the text handler is properly configured
	buf := &bytes.Buffer{}
	InitWithWriter(buf)

	slog.Info("test message", "key", "value")

	output := buf.String()
	// Verify it contains expected slog output format
	if !strings.Contains(output, "test message") {
		t.Errorf("expected message not found in output: %s", output)
	}
	if !strings.Contains(output, "key") {
		t.Errorf("expected key not found in output: %s", output)
	}
	if !strings.Contains(output, "value") {
		t.Errorf("expected value not found in output: %s", output)
	}
}
