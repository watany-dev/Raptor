package cli

import (
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

func TestParseRunFlags_IsolateFlag(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantIsolate bool
	}{
		{
			name:        "no isolate flag",
			args:        []string{"--workflow", "ci.yml", "--job", "build"},
			wantIsolate: false,
		},
		{
			name:        "long form --isolate",
			args:        []string{"--workflow", "ci.yml", "--job", "build", "--isolate"},
			wantIsolate: true,
		},
		{
			name:        "short form -i",
			args:        []string{"-w", "ci.yml", "-j", "build", "-i"},
			wantIsolate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := ParseRunFlags(tt.args)
			if err != nil {
				t.Fatalf("ParseRunFlags() error = %v", err)
			}
			if opts.Isolate != tt.wantIsolate {
				t.Errorf("Isolate = %v, want %v", opts.Isolate, tt.wantIsolate)
			}
		})
	}
}

func TestPrintHelp(t *testing.T) {
	// Just verify it doesn't panic
	PrintHelp()
}
