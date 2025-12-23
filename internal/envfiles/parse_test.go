package envfiles

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParseEnvFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected map[string]string
		wantErr  bool
	}{
		{
			name:     "empty file",
			content:  "",
			expected: map[string]string{},
			wantErr:  false,
		},
		{
			name:    "simple KEY=VALUE",
			content: "MY_VAR=hello",
			expected: map[string]string{
				"MY_VAR": "hello",
			},
			wantErr: false,
		},
		{
			name: "multiple KEY=VALUE pairs",
			content: `VAR1=value1
VAR2=value2
VAR3=value3`,
			expected: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
				"VAR3": "value3",
			},
			wantErr: false,
		},
		{
			name:    "value with equals sign",
			content: "CONNECTION_STRING=host=localhost;port=5432",
			expected: map[string]string{
				"CONNECTION_STRING": "host=localhost;port=5432",
			},
			wantErr: false,
		},
		{
			name:    "empty value",
			content: "EMPTY_VAR=",
			expected: map[string]string{
				"EMPTY_VAR": "",
			},
			wantErr: false,
		},
		{
			name: "multiline delimiter format",
			content: `MULTILINE<<EOF
line1
line2
line3
EOF`,
			expected: map[string]string{
				"MULTILINE": "line1\nline2\nline3",
			},
			wantErr: false,
		},
		{
			name: "multiline with custom delimiter",
			content: `JSON_DATA<<DELIM
{
  "key": "value"
}
DELIM`,
			expected: map[string]string{
				"JSON_DATA": "{\n  \"key\": \"value\"\n}",
			},
			wantErr: false,
		},
		{
			name: "mixed simple and multiline",
			content: `SIMPLE=value
MULTI<<END
multiline
content
END
ANOTHER=simple`,
			expected: map[string]string{
				"SIMPLE":  "value",
				"MULTI":   "multiline\ncontent",
				"ANOTHER": "simple",
			},
			wantErr: false,
		},
		{
			name: "skip empty lines",
			content: `VAR1=value1

VAR2=value2`,
			expected: map[string]string{
				"VAR1": "value1",
				"VAR2": "value2",
			},
			wantErr: false,
		},
		{
			name: "value with spaces",
			content: "MESSAGE=hello world",
			expected: map[string]string{
				"MESSAGE": "hello world",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			envFile := filepath.Join(tmpDir, "env")
			err := os.WriteFile(envFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			result, err := ParseEnvFile(envFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEnvFile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseEnvFile() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestParseEnvFile_NonExistentFile(t *testing.T) {
	result, err := ParseEnvFile("/nonexistent/path/file")
	if err != nil {
		t.Errorf("expected no error for non-existent file, got %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty map for non-existent file, got %v", result)
	}
}

func TestParseEnvFile_PermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping permission test as root")
	}

	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "env")
	err := os.WriteFile(envFile, []byte("VAR=value"), 0000)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = ParseEnvFile(envFile)
	if err == nil {
		t.Error("ParseEnvFile() expected error for permission denied, got nil")
	}
}

func TestParseEnvFile_InvalidEnvVarInMultiline(t *testing.T) {
	// Create a file with blocked env var (LD_PRELOAD) in multiline format
	content := `LD_PRELOAD<<EOF
line1
line2
EOF`
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, "env")
	err := os.WriteFile(envFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = ParseEnvFile(envFile)
	if err == nil {
		t.Error("ParseEnvFile() expected error for blocked env var LD_PRELOAD, got nil")
	}
}

func TestParsePathFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected []string
	}{
		{
			name:     "empty file",
			content:  "",
			expected: nil,
		},
		{
			name:     "single path",
			content:  "/usr/local/bin",
			expected: []string{"/usr/local/bin"},
		},
		{
			name: "multiple paths",
			content: `/usr/local/bin
/opt/bin
/home/user/.local/bin`,
			expected: []string{
				"/usr/local/bin",
				"/opt/bin",
				"/home/user/.local/bin",
			},
		},
		{
			name: "skip empty lines",
			content: `/first/path

/second/path`,
			expected: []string{
				"/first/path",
				"/second/path",
			},
		},
		{
			name: "trim whitespace",
			content: `  /path/with/spaces
/normal/path`,
			expected: []string{
				"/path/with/spaces",
				"/normal/path",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			pathFile := filepath.Join(tmpDir, "path")
			err := os.WriteFile(pathFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			result, err := ParsePathFile(pathFile)
			if err != nil {
				t.Errorf("ParsePathFile() unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParsePathFile() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestParsePathFile_NonExistentFile(t *testing.T) {
	result, err := ParsePathFile("/nonexistent/path/file")
	if err != nil {
		t.Errorf("expected no error for non-existent file, got %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice for non-existent file, got %v", result)
	}
}

func TestParsePathFile_PermissionDenied(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("skipping permission test as root")
	}

	tmpDir := t.TempDir()
	pathFile := filepath.Join(tmpDir, "path")
	err := os.WriteFile(pathFile, []byte("/usr/bin\n/bin"), 0000)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = ParsePathFile(pathFile)
	if err == nil {
		t.Error("ParsePathFile() expected error for permission denied, got nil")
	}
}

func TestPrependPath(t *testing.T) {
	tests := []struct {
		name        string
		currentPath string
		newPaths    []string
		expected    string
	}{
		{
			name:        "prepend single path",
			currentPath: "/usr/bin:/bin",
			newPaths:    []string{"/new/path"},
			expected:    "/new/path:/usr/bin:/bin",
		},
		{
			name:        "prepend multiple paths",
			currentPath: "/usr/bin",
			newPaths:    []string{"/first", "/second"},
			expected:    "/first:/second:/usr/bin",
		},
		{
			name:        "empty new paths",
			currentPath: "/usr/bin:/bin",
			newPaths:    []string{},
			expected:    "/usr/bin:/bin",
		},
		{
			name:        "empty current path",
			currentPath: "",
			newPaths:    []string{"/new/path"},
			expected:    "/new/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PrependPath(tt.currentPath, tt.newPaths)
			if result != tt.expected {
				t.Errorf("PrependPath() = %q, expected %q", result, tt.expected)
			}
		})
	}
}
