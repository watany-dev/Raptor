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

	// Shell command execution hooks
	"PROMPT_COMMAND": "executes arbitrary code before each bash command",

	// Dynamic linker (additional)
	"LD_AUDIT": "can inject audit libraries (same risk as LD_PRELOAD)",

	// Language startup hooks (code execution on interpreter start)
	"PYTHONSTARTUP": "executes arbitrary Python code on interpreter start",

	// Interactive command hijacking (used by git, less, etc.)
	"PAGER":  "can execute arbitrary commands instead of pager",
	"EDITOR": "can execute arbitrary commands instead of editor",
	"VISUAL": "can execute arbitrary commands instead of editor",
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

// SensitiveEnvVarPatterns contains patterns for environment variables that may contain secrets.
// These are filtered from inherited environment to prevent credential leakage.
var SensitiveEnvVarPatterns = []string{
	// AWS credentials
	"AWS_ACCESS_KEY_ID",
	"AWS_SECRET_ACCESS_KEY",
	"AWS_SESSION_TOKEN",
	"AWS_SECURITY_TOKEN",

	// Google Cloud
	"GOOGLE_APPLICATION_CREDENTIALS",
	"GOOGLE_CLOUD_PROJECT",
	"CLOUDSDK_AUTH_ACCESS_TOKEN",

	// Azure
	"AZURE_CLIENT_ID",
	"AZURE_CLIENT_SECRET",
	"AZURE_TENANT_ID",
	"AZURE_SUBSCRIPTION_ID",

	// Git/GitHub/GitLab
	"GITHUB_TOKEN",
	"GH_TOKEN",
	"GITLAB_TOKEN",
	"BITBUCKET_TOKEN",
	"GIT_ASKPASS",
	"GIT_CREDENTIAL_",

	// SSH/GPG agents
	"SSH_AUTH_SOCK",
	"SSH_AGENT_PID",
	"GPG_AGENT_INFO",
	"GPG_TTY",

	// Database credentials
	"DATABASE_URL",
	"DB_PASSWORD",
	"POSTGRES_PASSWORD",
	"MYSQL_PASSWORD",
	"REDIS_PASSWORD",
	"MONGODB_URI",

	// API keys and tokens (common patterns)
	"API_KEY",
	"API_SECRET",
	"AUTH_TOKEN",
	"ACCESS_TOKEN",
	"REFRESH_TOKEN",
	"BEARER_TOKEN",

	// NPM/Package managers
	"NPM_TOKEN",
	"NPM_AUTH_TOKEN",
	"YARN_AUTH_TOKEN",
	"NUGET_API_KEY",
	"PYPI_TOKEN",
	"RUBYGEMS_API_KEY",

	// Docker/Container registries
	"DOCKER_PASSWORD",
	"DOCKER_AUTH_CONFIG",
	"REGISTRY_PASSWORD",

	// CI/CD
	"CI_JOB_TOKEN",
	"CIRCLE_TOKEN",
	"TRAVIS_TOKEN",

	// Encryption keys
	"ENCRYPTION_KEY",
	"SIGNING_KEY",
	"PRIVATE_KEY",
	"SECRET_KEY",

	// General patterns (checked as suffix/contains)
	"_SECRET",
	"_PASSWORD",
	"_TOKEN",
	"_CREDENTIALS",
	"_API_KEY",
	"_PRIVATE_KEY",
}

// FilterSensitiveEnvVars filters out sensitive environment variables from the given list.
// It returns a new slice with only non-sensitive variables.
func FilterSensitiveEnvVars(envVars []string) []string {
	filtered := make([]string, 0, len(envVars))
	for _, env := range envVars {
		// Split into key=value
		idx := strings.Index(env, "=")
		if idx == -1 {
			continue // Invalid format, skip
		}
		key := strings.ToUpper(env[:idx])

		if !isSensitiveEnvVar(key) {
			filtered = append(filtered, env)
		}
	}
	return filtered
}

// isSensitiveEnvVar checks if an environment variable name matches sensitive patterns.
func isSensitiveEnvVar(name string) bool {
	upperName := strings.ToUpper(name)

	for _, pattern := range SensitiveEnvVarPatterns {
		// Exact match
		if upperName == pattern {
			return true
		}
		// Suffix match (for patterns like _SECRET, _PASSWORD)
		if strings.HasPrefix(pattern, "_") && strings.HasSuffix(upperName, pattern) {
			return true
		}
		// Prefix match (for patterns like GIT_CREDENTIAL_)
		if strings.HasSuffix(pattern, "_") && strings.HasPrefix(upperName, pattern) {
			return true
		}
	}
	return false
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
