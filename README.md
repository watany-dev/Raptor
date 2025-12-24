# Raptor

A lightweight CLI tool for running GitHub Actions workflows locally

## Overview

Raptor is a CLI tool for running GitHub Actions workflow files (`.github/workflows/*.yml`) in your local environment. Test your CI pipelines locally before pushing.

## Features

- Native support for GitHub Actions workflow YAML
- Environment variables support at workflow/job/step levels
- Dynamic environment variable propagation via `GITHUB_ENV` / `GITHUB_PATH`
- Per-step working directory configuration
- Lightweight and simple design

## Installation

### Download Binary (Recommended)

Download the binary for your platform from the [releases page](https://github.com/watany-dev/raptor/releases).

#### Linux (x86_64)

```bash
curl -LO https://github.com/watany-dev/raptor/releases/download/v0.1.2/raptor_0.1.2_Linux_x86_64.tar.gz
tar xzf raptor_0.1.2_Linux_x86_64.tar.gz
sudo mv raptor /usr/local/bin/
raptor --version
```

#### Linux (ARM64)

```bash
curl -LO https://github.com/watany-dev/raptor/releases/download/v0.1.2/raptor_0.1.2_Linux_arm64.tar.gz
tar xzf raptor_0.1.2_Linux_arm64.tar.gz
sudo mv raptor /usr/local/bin/
raptor --version
```

#### macOS (Apple Silicon)

```bash
curl -LO https://github.com/watany-dev/raptor/releases/download/v0.1.2/raptor_0.1.2_Darwin_arm64.tar.gz
tar xzf raptor_0.1.2_Darwin_arm64.tar.gz
sudo mv raptor /usr/local/bin/
raptor --version
```

#### macOS (Intel)

```bash
curl -LO https://github.com/watany-dev/raptor/releases/download/v0.1.2/raptor_0.1.2_Darwin_x86_64.tar.gz
tar xzf raptor_0.1.2_Darwin_x86_64.tar.gz
sudo mv raptor /usr/local/bin/
raptor --version
```

#### Windows (x86_64)

1. Download [raptor_0.1.2_Windows_x86_64.zip](https://github.com/watany-dev/raptor/releases/download/v0.1.2/raptor_0.1.2_Windows_x86_64.zip)
2. Extract the ZIP file
3. Place `raptor.exe` in a directory in your PATH

### Go install

```bash
go install github.com/watany-dev/raptor/cmd/raptor@v0.1.2
```

### Build from Source

```bash
git clone https://github.com/watany-dev/raptor.git
cd raptor
go build -o raptor ./cmd/raptor
sudo mv raptor /usr/local/bin/
```

### Requirements

- Runtime: Git 2.5 or later
- Build from source only: Go 1.24 or later

## Usage

### Basic Usage

```bash
# Run a specific job
raptor run --workflow <workflow-file> --job <job-id>

# Run all jobs (omit --job)
raptor run --workflow <workflow-file>
```

### Command Options

| Option | Short | Description |
|--------|-------|-------------|
| `--workflow` | `-w` | Path to workflow file (required) |
| `--job` | `-j` | Job ID to run (runs all jobs if omitted) |
| `--workdir` | `-C` | Working directory (default: current directory) |
| `--dry-run` | `-n` | Preview without actually executing |

**Note**: All workflows are executed in isolated Git worktrees for security.

### Dry-run Mode

Dry-run mode allows you to preview what will be executed without actually running the workflow. Useful for checking and debugging CI pipelines.

```bash
# Explicitly specify dry-run flag
raptor run -w ci.yml --dry-run
raptor run -w ci.yml -n

# Omit run subcommand for automatic dry-run mode
raptor -w ci.yml
```

Example dry-run output:

```
🔍 DRY RUN MODE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Workflow: .github/workflows/ci.yml
Name: CI
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

📋 Job: build
   Runs-on: ubuntu-latest

   [1] Setup
       Command:
         echo "Setting up..."

   [2] Build
       Command:
         echo "Building..."

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
To execute this workflow, use: raptor run -w .github/workflows/ci.yml
```

Information displayed:
- Workflow name and path
- Job ID, name, and runs-on
- Each step's name, working directory, conditions (`if`), number of environment variables, and command content

### Examples

```bash
# Run the build job from CI workflow
raptor run --workflow .github/workflows/ci.yml --job build

# Use short form
raptor run -w ci.yml -j test

# Run all jobs in workflow
raptor run -w ci.yml

# Run with specified working directory
raptor run -w .github/workflows/ci.yml -j lint -C /path/to/project
```

### Help

```bash
raptor help
raptor --help
```

### Check Version

```bash
raptor version
raptor --version
raptor -v
```

## Supported Features

### Workflow Syntax

Currently supported GitHub Actions syntax:

| Feature | Support |
|---------|---------|
| `name` (workflow/job/step names) | ✅ |
| `env` (environment variables) | ✅ |
| `run` (shell commands) | ✅ |
| `working-directory` | ✅ |
| `GITHUB_ENV` | ✅ |
| `GITHUB_PATH` | ✅ |
| `if` (conditionals) | ✅ (full support with AND/OR/NOT, string functions, hashFiles) |
| `uses` (actions) | ❌ |
| `with` (action inputs) | ❌ |
| `matrix` (matrix builds) | ❌ |

### Conditionals (`if`)

Conditional step execution is supported:

```yaml
steps:
  - name: Always run
    if: true
    run: echo "This always runs"

  - name: Conditional
    if: ${{ env.MY_VAR == 'value' }}
    run: echo "Runs when MY_VAR is 'value'"

  - name: On failure
    if: failure()
    run: echo "Runs only if previous step failed"

  - name: Always (even on failure)
    if: always()
    run: echo "Cleanup step"

  - name: Complex conditions with AND/OR
    if: ${{ success() && env.DEPLOY_ENV == 'production' }}
    run: echo "Deploy to production"

  - name: String functions
    if: ${{ startsWith(env.BRANCH_NAME, 'feature/') }}
    run: echo "Feature branch detected"

  - name: File hash for caching
    if: ${{ hashFiles('package.json') != '' }}
    run: echo "package.json exists"
```

**Supported conditional syntax:**

| Syntax | Description |
|--------|-------------|
| `true` / `false` | Literal boolean values |
| `success()` | All previous steps succeeded |
| `failure()` | Any step failed |
| `always()` | Always execute (continue after failure) |
| `cancelled()` | Execute on cancellation (always false) |
| `${{ env.VAR == 'value' }}` | Environment variable comparison |
| `${{ env.VAR != 'value' }}` | Environment variable negation |
| `${{ steps.ID.outcome == 'success' }}` | Step result reference |
| `${{ expr1 && expr2 }}` | Logical AND operator |
| `${{ expr1 \|\| expr2 }}` | Logical OR operator |
| `${{ !expr }}` | Logical NOT operator |
| `${{ (expr) }}` | Grouping with parentheses |
| `contains(search, item)` | Check if string/array contains value |
| `startsWith(search, prefix)` | Check if string starts with prefix |
| `endsWith(search, suffix)` | Check if string ends with suffix |
| `hashFiles(pattern, ...)` | Calculate SHA-256 hash of files matching patterns |

### Sample Workflow

Example workflow that can be run with Raptor:

```yaml
name: CI

env:
  GLOBAL_VAR: "global-value"

jobs:
  build:
    runs-on: ubuntu-latest
    env:
      JOB_VAR: "job-value"
    steps:
      - name: Setup
        run: echo "Setting up..."
        env:
          STEP_VAR: "step-value"

      - name: Build
        run: |
          echo "Building..."
          echo "GLOBAL_VAR=$GLOBAL_VAR"

      - name: Set dynamic env
        run: |
          echo "MY_VAR=dynamic-value" >> $GITHUB_ENV

      - name: Use dynamic env
        run: echo "MY_VAR is $MY_VAR"
```

## Security

Since Raptor executes commands described in workflow files, **only run trusted workflows**.

### Security Features

- **Isolated execution**: All workflows run in isolated Git worktrees
- **Absolute path prohibition**: Absolute paths cannot be used in `working-directory`
- **Environment variable protection**: Dangerous environment variables like `LD_PRELOAD` are blocked
- **Input validation**: Environment variable names and values are validated

See [SECURITY.md](SECURITY.md) for details.

### Warnings

Raptor runs workflows **with your user permissions**. Malicious workflows can:

- Delete or modify files
- Access network
- Send data externally

**Always verify the content before running.**

## Development

### Build

```bash
go build ./...
```

### Run Tests

```bash
go test ./...
```

### Test Coverage

```bash
go test -cover ./...
```

## Project Structure

```
raptor/
├── cmd/raptor/        # CLI entry point
├── internal/
│   ├── cli/           # CLI flag parsing and runner
│   ├── envfiles/      # GITHUB_ENV/GITHUB_PATH parsing
│   ├── executor/      # Command execution engine
│   ├── expression/    # Expression evaluation (if conditions)
│   ├── runtime/       # Environment variable merging
│   ├── security/      # Security validation
│   ├── util/          # Git operation utilities
│   ├── workflow/      # Workflow YAML parsing
│   └── worktree/      # Git worktree management
└── docs/              # Development documentation
```

## License

Apache License 2.0
