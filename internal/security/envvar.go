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
	"LD_AUDIT":              "can inject audit libraries",
	"LD_DEBUG":              "can leak sensitive information",
	"DYLD_INSERT_LIBRARIES": "macOS library injection",
	"DYLD_LIBRARY_PATH":     "macOS library path hijacking",
	"DYLD_FRAMEWORK_PATH":   "macOS framework hijacking",

	// Shell behavior modification
	"ENV":            "can execute arbitrary code on shell start",
	"BASH_ENV":       "can execute arbitrary code on bash start",
	"PROMPT_COMMAND": "can execute arbitrary code before each prompt",
	"PS4":            "can execute code via debug mode expansion",
	"SHELLOPTS":      "can modify shell behavior",
	"BASHOPTS":       "can modify bash behavior",
	"CDPATH":         "can alter directory resolution",

	// Security-sensitive
	"IFS":        "can break command parsing",
	"GLOBIGNORE": "can alter file globbing behavior",

	// Git-specific (protect git operations)
	"GIT_DIR":              "can redirect git operations",
	"GIT_WORK_TREE":        "can redirect git work tree",
	"GIT_INDEX_FILE":       "can corrupt git index",
	"GIT_OBJECT_DIRECTORY": "can redirect git objects",
	"GIT_SSH_COMMAND":      "can hijack git SSH operations",
	"GIT_ASKPASS":          "can hijack git credential prompts",
	"GIT_EXEC_PATH":        "can redirect git executables",
	"GIT_TEMPLATE_DIR":     "can inject git hooks via templates",
	"GIT_PAGER":            "can execute arbitrary commands as pager",
	"GIT_EDITOR":           "can execute arbitrary commands as editor",
	"GIT_PROXY_COMMAND":    "can intercept git network operations",

	// Programming language injection vectors
	"NODE_OPTIONS":     "can inject Node.js modules via --require",
	"NODE_PATH":        "can redirect Node.js module loading",
	"PYTHONPATH":       "can inject Python modules",
	"PYTHONSTARTUP":    "can execute code on Python start",
	"PYTHONHOME":       "can redirect Python installation",
	"RUBYOPT":          "can inject Ruby options/modules",
	"RUBYLIB":          "can redirect Ruby library loading",
	"PERL5LIB":         "can inject Perl modules",
	"PERL5OPT":         "can inject Perl options",
	"JAVA_TOOL_OPTIONS": "can inject Java agents/options",
	"_JAVA_OPTIONS":    "can inject Java options",
	"CLASSPATH":        "can redirect Java class loading",

	// Network proxies (MITM attack vectors)
	"HTTP_PROXY":   "can redirect HTTP traffic through malicious proxy",
	"HTTPS_PROXY":  "can redirect HTTPS traffic through malicious proxy",
	"FTP_PROXY":    "can redirect FTP traffic through malicious proxy",
	"ALL_PROXY":    "can redirect all traffic through malicious proxy",
	"NO_PROXY":     "can selectively bypass proxy security",
	"SSL_CERT_DIR": "can redirect SSL certificate validation",
	"SSL_CERT_FILE": "can redirect SSL certificate validation",
	"CURL_CA_BUNDLE": "can redirect curl certificate validation",
	"REQUESTS_CA_BUNDLE": "can redirect Python requests certificate validation",

	// System path hijacking
	"HOME":   "can redirect home directory and config files",
	"TMPDIR": "can redirect temporary file operations",
	"TMP":    "can redirect temporary file operations",
	"TEMP":   "can redirect temporary file operations",
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
