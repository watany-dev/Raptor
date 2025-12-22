# Security Policy

## Security Model

Raptor executes GitHub Actions workflows locally on your machine. **The security model assumes that you trust the workflow files you execute.**

### Threat Model

**What Raptor protects against:**
- Accidental repository corruption (via isolated git worktrees)
- Malicious environment variable injection (LD_PRELOAD, BASH_ENV, etc.)
- Path traversal outside the workspace
- Absolute path manipulation

**What Raptor DOES NOT protect against:**
- Malicious commands in trusted workflow files
- Network-based attacks from workflow commands
- Resource exhaustion (CPU, memory, disk)

### Best Practices

1. **Only run workflows from trusted sources**
   - Review workflow files before execution
   - Use version control to track changes

2. **Use isolated mode (default)**
   - Raptor runs in isolated git worktrees by default
   - This protects your main repository from corruption

3. **Review environment variables**
   - Raptor blocks dangerous environment variables
   - See blocked list in `internal/security/envvar.go`

4. **Monitor execution**
   - Watch command output for suspicious activity
   - Check resource usage during execution

## Security Features

### 1. Isolated Execution (Default)
- All workflows run in isolated git worktrees
- Main repository is protected from modifications
- Automatic cleanup after execution

### 2. Path Restrictions
- Absolute paths are blocked in `working-directory`
- Path traversal outside workspace is prevented
- Relative paths are enforced

### 3. Environment Variable Protection
Blocked variables:
- `LD_PRELOAD`, `LD_LIBRARY_PATH` (library injection)
- `DYLD_INSERT_LIBRARIES`, `DYLD_LIBRARY_PATH` (macOS library injection)
- `BASH_ENV`, `ENV` (shell startup scripts)
- `IFS`, `GLOBIGNORE` (shell behavior modification)
- `GIT_DIR`, `GIT_WORK_TREE`, `GIT_INDEX_FILE`, `GIT_OBJECT_DIRECTORY` (git redirection)

### 4. Input Validation
- Environment variable names: `[A-Za-z_][A-Za-z0-9_]*`
- Environment variable values: max 100KB, no null bytes
- Working directories: relative paths only

## Reporting Vulnerabilities

If you discover a security vulnerability in Raptor, please report it to:
- **GitHub Security Advisories**: [Create advisory](https://github.com/watany-dev/raptor/security/advisories/new)

Please include:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

We will respond within 48 hours and work on a fix.

## Changelog

### Security Enhancements (Current)
- Made isolated worktree execution mandatory
- Blocked absolute paths in working-directory
- Added environment variable blocklist
- Improved security warnings

## Acknowledgments

We thank the security community for their contributions to making Raptor safer.
