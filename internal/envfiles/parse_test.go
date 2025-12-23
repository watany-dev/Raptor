package envfiles

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
			name:    "value with spaces",
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

			if !maps.Equal(result, tt.expected) {
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

			if !slices.Equal(result, tt.expected) {
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

// TestParseEnvFile_UnclosedHeredoc tests handling of unclosed heredoc delimiters
// When a heredoc delimiter is never closed, the parser reads until EOF and uses all remaining content as the value
func TestParseEnvFile_UnclosedHeredoc(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	content := `VAR1=simple
VAR2<<EOF
multiline content
but missing EOF line`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	result, err := ParseEnvFile(envFile)
	if err != nil {
		t.Fatalf("ParseEnvFile() error = %v", err)
	}

	// VAR1 should be parsed correctly
	if result["VAR1"] != "simple" {
		t.Errorf("VAR1 = %q, want %q", result["VAR1"], "simple")
	}

	// VAR2 should contain all remaining content until EOF (unclosed heredoc behavior)
	expectedVAR2 := "multiline content\nbut missing EOF line"
	if result["VAR2"] != expectedVAR2 {
		t.Errorf("VAR2 = %q, want %q", result["VAR2"], expectedVAR2)
	}
}

// TestParseEnvFile_LongValue tests handling of long variable values
func TestParseEnvFile_LongValue(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	// Create a moderately long value (10KB, under scanner limits)
	longValue := strings.Repeat("x", 10*1024)
	content := fmt.Sprintf("LONG_VAR=%s\nSHORT_VAR=short", longValue)

	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	result, err := ParseEnvFile(envFile)
	if err != nil {
		t.Fatalf("ParseEnvFile() error = %v", err)
	}

	if val, exists := result["LONG_VAR"]; !exists || len(val) != len(longValue) {
		t.Errorf("ParseEnvFile() failed to parse long value correctly")
	}
	if val, exists := result["SHORT_VAR"]; !exists || val != "short" {
		t.Errorf("ParseEnvFile() failed to parse SHORT_VAR correctly")
	}
}

// TestParseEnvFile_MultilineHeredocs tests multiple heredoc blocks in sequence
func TestParseEnvFile_MultilineHeredocs(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	content := `
VAR1=simple
VAR2<<EOF
first multiline
value here
EOF
VAR3<<END
second multiline
value here
END
VAR4=final
`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	result, err := ParseEnvFile(envFile)
	if err != nil {
		t.Fatalf("ParseEnvFile() error = %v", err)
	}

	if _, exists := result["VAR1"]; !exists {
		t.Error("ParseEnvFile() failed to parse VAR1")
	}
	if _, exists := result["VAR2"]; !exists {
		t.Error("ParseEnvFile() failed to parse VAR2 (first heredoc)")
	}
	if _, exists := result["VAR3"]; !exists {
		t.Error("ParseEnvFile() failed to parse VAR3 (second heredoc)")
	}
	if _, exists := result["VAR4"]; !exists {
		t.Error("ParseEnvFile() failed to parse VAR4")
	}
}

// TestParsePathFile_EdgeCases tests edge cases for path file parsing
func TestParsePathFile_EdgeCases(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	pathFile := filepath.Join(tmpDir, "path")

	content := `
/first/path
/second/path

/fourth/path
`
	if err := os.WriteFile(pathFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write path file: %v", err)
	}

	result, err := ParsePathFile(pathFile)
	if err != nil {
		t.Fatalf("ParsePathFile() error = %v", err)
	}

	if len(result) < 3 {
		t.Errorf("ParsePathFile() expected at least 3 paths, got %d", len(result))
	}
}

// TestParsePathFile_SpecialCharacters tests paths with special characters
func TestParsePathFile_SpecialCharacters(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	pathFile := filepath.Join(tmpDir, "path")

	content := `/path/with spaces/bin
/path/with-dashes/bin
/path/with_underscores/bin
/path/with.dots/bin`

	if err := os.WriteFile(pathFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write path file: %v", err)
	}

	result, err := ParsePathFile(pathFile)
	if err != nil {
		t.Fatalf("ParsePathFile() error = %v", err)
	}

	if len(result) != 4 {
		t.Errorf("ParsePathFile() expected 4 paths, got %d", len(result))
	}

	// Verify paths are properly parsed
	pathStr := strings.Join(result, ":")
	if !strings.Contains(pathStr, "/path/with spaces/bin") {
		t.Error("ParsePathFile() failed to preserve spaces in path")
	}
}

// TestParseEnvFile_InvalidEnvVarName tests handling of invalid env var names
func TestParseEnvFile_InvalidEnvVarName(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	// Test with blocked env var name in simple format
	content := `LD_PRELOAD=/malicious/path`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	_, err := ParseEnvFile(envFile)
	if err == nil {
		t.Error("ParseEnvFile() expected error for blocked env var LD_PRELOAD")
	}
}

// TestParseEnvFile_HeredocValueValidation tests heredoc value validation
func TestParseEnvFile_HeredocValueValidation(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	// Test with valid heredoc value
	content := `NORMAL_VAR<<EOF
just some text
with multiple lines
EOF`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	result, err := ParseEnvFile(envFile)
	if err != nil {
		t.Fatalf("ParseEnvFile() error = %v", err)
	}

	expected := "just some text\nwith multiple lines"
	if result["NORMAL_VAR"] != expected {
		t.Errorf("NORMAL_VAR = %q, want %q", result["NORMAL_VAR"], expected)
	}
}

// TestParseEnvFile_LineWithoutEquals tests handling of lines without equals sign
func TestParseEnvFile_LineWithoutEquals(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	// Line without equals sign should be ignored
	content := `VAR1=value1
just some text without equals
VAR2=value2`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	result, err := ParseEnvFile(envFile)
	if err != nil {
		t.Fatalf("ParseEnvFile() error = %v", err)
	}

	if result["VAR1"] != "value1" {
		t.Errorf("VAR1 = %q, want %q", result["VAR1"], "value1")
	}
	if result["VAR2"] != "value2" {
		t.Errorf("VAR2 = %q, want %q", result["VAR2"], "value2")
	}
	// The line without equals should not create any entry
	if len(result) != 2 {
		t.Errorf("Expected 2 entries, got %d", len(result))
	}
}

// TestParseEnvFile_WhitespaceHandling tests whitespace handling
func TestParseEnvFile_WhitespaceHandling(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	// Test whitespace handling around key names
	content := `  SPACES_BEFORE  =value
TRAILING_SPACES=value
  BOTH_SIDES  =value`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	result, err := ParseEnvFile(envFile)
	if err != nil {
		t.Fatalf("ParseEnvFile() error = %v", err)
	}

	// Key should have spaces trimmed
	if _, ok := result["SPACES_BEFORE"]; !ok {
		t.Error("SPACES_BEFORE should be present (with trimmed key)")
	}
}

// TestPrependPath_EmptyNewPaths tests PrependPath with empty new paths
func TestPrependPath_EmptyNewPaths(t *testing.T) {
	result := PrependPath("/usr/bin:/bin", []string{})
	if result != "/usr/bin:/bin" {
		t.Errorf("PrependPath with empty new paths = %q, want %q", result, "/usr/bin:/bin")
	}
}

// TestPrependPath_EmptyCurrentPath tests PrependPath with empty current path
func TestPrependPath_EmptyCurrentPath(t *testing.T) {
	result := PrependPath("", []string{"/new/path"})
	if result != "/new/path" {
		t.Errorf("PrependPath with empty current path = %q, want %q", result, "/new/path")
	}
}

// TestPrependPath_MultipleNewPaths tests PrependPath with multiple new paths
func TestPrependPath_MultipleNewPaths(t *testing.T) {
	result := PrependPath("/existing", []string{"/first", "/second", "/third"})
	expected := "/first:/second:/third:/existing"
	if result != expected {
		t.Errorf("PrependPath = %q, want %q", result, expected)
	}
}

// TestParseEnvFile_InvalidEnvVarValueSimpleFormat tests value validation error in simple KEY=VALUE format
func TestParseEnvFile_InvalidEnvVarValueSimpleFormat(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	// Test with blocked env var value in simple format
	// Note: The security validation checks the key name, not the value content
	// So we need to use a blocked env var name
	content := `LD_LIBRARY_PATH=/malicious/path`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	_, err := ParseEnvFile(envFile)
	if err == nil {
		t.Error("ParseEnvFile() expected error for blocked env var LD_LIBRARY_PATH")
	}
}

// TestParseEnvFile_HeredocInvalidValue tests heredoc value validation failure
func TestParseEnvFile_HeredocInvalidValue(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	// Create a file with extremely long value that could trigger validation
	// Using a value that exceeds reasonable limits
	content := `DYLD_INSERT_LIBRARIES<<EOF
/some/path
EOF`
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	_, err := ParseEnvFile(envFile)
	if err == nil {
		t.Error("ParseEnvFile() expected error for blocked env var DYLD_INSERT_LIBRARIES")
	}
}

// TestParseEnvFile_HeredocValueWithNullByte tests heredoc value validation with null byte
// This specifically covers the ValidateEnvVarValue error path (line 64-66)
func TestParseEnvFile_HeredocValueWithNullByte(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	// Create a file with heredoc containing null byte in value
	// The key is valid, but the value contains a null byte which should fail validation
	content := "VALID_VAR<<EOF\nvalue with \x00 null byte\nEOF"
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	_, err := ParseEnvFile(envFile)
	if err == nil {
		t.Error("ParseEnvFile() expected error for heredoc value containing null byte")
	}
}

// TestParsePathFile_ScannerError tests the scanner.Err() path by creating a file
// with a line that exceeds the default scanner buffer size
func TestParsePathFile_ScannerError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	pathFile := filepath.Join(tmpDir, "path")

	// Create a file with a very long line (> 64KB to exceed default scanner buffer)
	longLine := strings.Repeat("a", 70000)
	content := "/usr/bin\n" + longLine + "\n/bin"
	if err := os.WriteFile(pathFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write path file: %v", err)
	}

	_, err := ParsePathFile(pathFile)
	// Should return an error because the line is too long for the scanner
	if err == nil {
		t.Error("ParsePathFile() expected error for line exceeding scanner buffer")
	}
}

// TestParseEnvFile_ScannerError tests the scanner.Err() path for env files
func TestParseEnvFile_ScannerError(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")

	// Create a file with a very long line (> 64KB to exceed default scanner buffer)
	longLine := strings.Repeat("a", 70000)
	content := "VAR1=value1\n" + longLine + "\nVAR2=value2"
	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write env file: %v", err)
	}

	_, err := ParseEnvFile(envFile)
	// Should return an error because the line is too long for the scanner
	if err == nil {
		t.Error("ParseEnvFile() expected error for line exceeding scanner buffer")
	}
}
