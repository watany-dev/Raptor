package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseRunFlags_WorkflowFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		wantPath string
	}{
		{
			name:     "long form --workflow",
			args:     []string{"--workflow", "ci.yml", "--job", "build"},
			wantPath: "ci.yml",
		},
		{
			name:     "short form -w",
			args:     []string{"-w", "ci.yml", "-j", "build"},
			wantPath: "ci.yml",
		},
		{
			name:     "full path",
			args:     []string{"--workflow", ".github/workflows/test.yml", "--job", "test"},
			wantPath: ".github/workflows/test.yml",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseRunFlags(tt.args)
			if err != nil {
				t.Fatalf("ParseRunFlags() error = %v", err)
			}
			if opts.Workflow != tt.wantPath {
				t.Errorf("Workflow = %v, want %v", opts.Workflow, tt.wantPath)
			}
		})
	}
}

func TestParseRunFlags_JobFlag(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantJob string
	}{
		{
			name:    "long form --job",
			args:    []string{"--workflow", "ci.yml", "--job", "build"},
			wantJob: "build",
		},
		{
			name:    "short form -j",
			args:    []string{"-w", "ci.yml", "-j", "test"},
			wantJob: "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseRunFlags(tt.args)
			if err != nil {
				t.Fatalf("ParseRunFlags() error = %v", err)
			}
			if opts.Job != tt.wantJob {
				t.Errorf("Job = %v, want %v", opts.Job, tt.wantJob)
			}
		})
	}
}

func TestParseRunFlags_MissingRequired(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{
			name:    "missing workflow",
			args:    []string{"--job", "build"},
			wantErr: "--workflow flag is required",
		},
		{
			name:    "both missing",
			args:    []string{},
			wantErr: "--workflow flag is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseRunFlags(tt.args)
			if err == nil {
				t.Fatal("ParseRunFlags() expected error, got nil")
			}
			if err.Error() != tt.wantErr {
				t.Errorf("error = %v, want %v", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestParseRunFlags_OptionalJob(t *testing.T) {
	// Job is optional - when omitted, all jobs should run
	opts, err := ParseRunFlags([]string{"--workflow", "ci.yml"})
	if err != nil {
		t.Fatalf("ParseRunFlags() error = %v", err)
	}
	if opts.Job != "" {
		t.Errorf("Job = %v, want empty string", opts.Job)
	}
	if opts.Workflow != "ci.yml" {
		t.Errorf("Workflow = %v, want ci.yml", opts.Workflow)
	}
}

func TestParseRunFlags_WorkDir(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantDir string
	}{
		{
			name:    "long form --workdir",
			args:    []string{"--workflow", "ci.yml", "--job", "build", "--workdir", "/tmp/test"},
			wantDir: "/tmp/test",
		},
		{
			name:    "short form -C",
			args:    []string{"-w", "ci.yml", "-j", "build", "-C", "/home/user"},
			wantDir: "/home/user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseRunFlags(tt.args)
			if err != nil {
				t.Fatalf("ParseRunFlags() error = %v", err)
			}
			if opts.WorkingDir != tt.wantDir {
				t.Errorf("WorkingDir = %v, want %v", opts.WorkingDir, tt.wantDir)
			}
		})
	}
}

func TestParseRunFlags_DefaultWorkDir(t *testing.T) {
	opts, err := ParseRunFlags([]string{"--workflow", "ci.yml", "--job", "build"})
	if err != nil {
		t.Fatalf("ParseRunFlags() error = %v", err)
	}
	if opts.WorkingDir == "" {
		t.Error("WorkingDir should have a default value")
	}
}

func TestPrintHelp(t *testing.T) {
	// Test that PrintHelp runs without panic (uses stdout)
	PrintHelp()
}

func TestFprintHelp(t *testing.T) {
	var buf bytes.Buffer
	FprintHelp(&buf)
	output := buf.String()

	// Verify essential help content is present
	requiredContent := []string{
		"Usage:",
		"raptor",
		"Commands:",
		"run",
		"Options:",
		"--workflow",
		"--job",
		"--workdir",
		"--dry-run",
		"--ignore-if-errors",
		"Examples:",
		"Security:",
	}

	for _, content := range requiredContent {
		if !strings.Contains(output, content) {
			t.Errorf("Help output missing required content: %q", content)
		}
	}

	// Verify output is not empty and has reasonable length
	if len(output) < 500 {
		t.Errorf("Help output seems too short (%d bytes), expected comprehensive help", len(output))
	}
}

// TestParseRunFlags_DryRunFlag tests the dry-run flag
func TestParseRunFlags_DryRunFlag(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantDryRun bool
	}{
		{
			name:       "long form --dry-run",
			args:       []string{"--workflow", "ci.yml", "--dry-run"},
			wantDryRun: true,
		},
		{
			name:       "short form -n",
			args:       []string{"-w", "ci.yml", "-n"},
			wantDryRun: true,
		},
		{
			name:       "without dry-run",
			args:       []string{"-w", "ci.yml"},
			wantDryRun: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseRunFlags(tt.args)
			if err != nil {
				t.Fatalf("ParseRunFlags() error = %v", err)
			}
			if opts.DryRun != tt.wantDryRun {
				t.Errorf("DryRun = %v, want %v", opts.DryRun, tt.wantDryRun)
			}
		})
	}
}

// TestParseRunFlags_FlagParseError tests flag parsing errors
func TestParseRunFlags_FlagParseError(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "invalid flag",
			args: []string{"--invalid-flag-that-does-not-exist"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseRunFlags(tt.args)
			if err == nil {
				t.Error("ParseRunFlags() expected error for invalid flag")
			}
		})
	}
}

// TestParseRunFlags_IgnoreIfErrors tests the --ignore-if-errors flag
func TestParseRunFlags_IgnoreIfErrors(t *testing.T) {
	tests := []struct {
		name               string
		args               []string
		wantIgnoreIfErrors bool
	}{
		{
			name:               "with --ignore-if-errors",
			args:               []string{"--workflow", "ci.yml", "--ignore-if-errors"},
			wantIgnoreIfErrors: true,
		},
		{
			name:               "without --ignore-if-errors",
			args:               []string{"-w", "ci.yml"},
			wantIgnoreIfErrors: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseRunFlags(tt.args)
			if err != nil {
				t.Fatalf("ParseRunFlags() error = %v", err)
			}
			if opts.IgnoreIfErrors != tt.wantIgnoreIfErrors {
				t.Errorf("IgnoreIfErrors = %v, want %v", opts.IgnoreIfErrors, tt.wantIgnoreIfErrors)
			}
		})
	}
}
