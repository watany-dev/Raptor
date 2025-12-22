package security

import (
	"fmt"
	"regexp"
	"strings"
)

// BlockedEnvVars contains environment variables that should never be set by workflows.
var BlockedEnvVars = map[string]string{
	// Dynamic linker hijacking
	"LD_PRELOAD":            "can inject malicious libraries",
	"LD_LIBRARY_PATH":       "can redirect library loading",
	"DYLD_INSERT_LIBRARIES": "macOS library injection",
	"DYLD_LIBRARY_PATH":     "macOS library path hijacking",

	// Shell behavior modification
	"ENV":      "can execute arbitrary code on shell start",
	"BASH_ENV": "can execute arbitrary code on bash start",

	// Security-sensitive
	"IFS":        "can break command parsing",
	"GLOBIGNORE": "can alter file globbing behavior",

	// Git-specific (protect git operations)
	"GIT_DIR":              "can redirect git operations",
	"GIT_WORK_TREE":        "can redirect git work tree",
	"GIT_INDEX_FILE":       "can corrupt git index",
	"GIT_OBJECT_DIRECTORY": "can redirect git objects",
}

// validEnvVarName matches valid environment variable names (A-Z, a-z, 0-9, _ starting with letter or underscore)
var validEnvVarName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// ValidateEnvVarName validates that an environment variable name is safe.
func ValidateEnvVarName(name string) error {
	// 1. Empty check
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("environment variable name cannot be empty")
	}

	// 2. Block list check (case insensitive)
	upperName := strings.ToUpper(name)
	if reason, blocked := BlockedEnvVars[upperName]; blocked {
		return fmt.Errorf(
			"environment variable %q is blocked for security: %s",
			name, reason,
		)
	}

	// 3. Format check (A-Z, a-z, 0-9, _ only)
	if !validEnvVarName.MatchString(name) {
		return fmt.Errorf(
			"invalid environment variable name %q: must start with letter or underscore, "+
				"and contain only letters, numbers, and underscores",
			name,
		)
	}

	return nil
}

// ValidateEnvVarValue validates that an environment variable value is safe.
func ValidateEnvVarValue(name, value string) error {
	// Value length limit (DoS prevention)
	const maxValueLength = 1024 * 100 // 100KB
	if len(value) > maxValueLength {
		return fmt.Errorf(
			"environment variable %q value is too long: %d bytes (max: %d)",
			name, len(value), maxValueLength,
		)
	}

	// NULL byte check
	if strings.Contains(value, "\x00") {
		return fmt.Errorf(
			"environment variable %q contains null bytes",
			name,
		)
	}

	return nil
}
