The plan includes eight comprehensive sections

1) Overview: The goal and high-level approach
Goal
- Build a lightweight Go tool that runs a subset of GitHub Actions locally in an isolated Git worktree (“workspace”), without depending on act.

High-level approach
1. Create an isolated workspace using `git worktree add --detach` under a tool-owned directory (e.g., `.garl/ws-<id>`).
2. Load a workflow YAML from `.github/workflows/*.yml`, select a workflow/job to run.
3. Execute a job as an ordered list of steps:
   - MVP: `run:` steps only
   - Apply env layering: workflow → job → step
   - Provide minimal default env: GITHUB_WORKSPACE, GITHUB_ACTIONS=true, CI=true, GITHUB_SHA, GITHUB_REF, etc.
4. Support environment files to propagate state between steps:
   - GITHUB_ENV (exports for subsequent steps)
   - GITHUB_PATH (PATH augmentation)
   - GITHUB_OUTPUT (step outputs; optional for MVP but recommended early)
5. Keep the design extensible:
   - Executors: HostExecutor now, optional DockerExecutor later
   - Action resolver (uses:) explicitly out-of-scope for MVP, but architecture leaves a hook.

------------------------------------------------------------
2) Types: Complete type definitions and data structures

- internal/workflow/model.go

WorkflowFile
- Name string
- On map[string]any              (MVP: parse but don’t deeply interpret)
- Env map[string]string
- Jobs map[string]Job

Job
- Name string
- RunsOn any                     (string or []string; MVP: ignore or profile-map later)
- Needs any                      (MVP: ignore)
- Env map[string]string
- Steps []Step

Step
- Name string
- ID string                      (for outputs; recommended early)
- If string                      (MVP: ignored)
- Shell string
- WorkingDirectory string
- Env map[string]string
- Run string                     (MVP supports Run only)
- Uses string                    (MVP: reject clearly)
- With map[string]any

JobPlan
- WorkflowPath string
- JobID string
- Job Job
- Steps []StepPlan

StepPlan
- Index int
- Step Step
- Shell string                   (resolved)
- WorkingDirectory string        (resolved)
- Env map[string]string          (resolved)

- internal/runtime/state.go

State
- BaseEnv map[string]string                      (persisted across steps)
- StepOutputs map[string]map[string]string       (stepID -> outputName -> value)
- PathPrepend []string                           (from GITHUB_PATH)

- internal/envfiles/types.go

StepFiles
- EnvFile string        (GITHUB_ENV)
- PathFile string       (GITHUB_PATH)
- OutputFile string     (GITHUB_OUTPUT)
- SummaryFile string    (GITHUB_STEP_SUMMARY; optional)
- TempDir string

- internal/executor/executor.go

ExecSpec
- Dir string
- Env []string
- Shell string
- Script string

Result
- ExitCode int

Executor interface
- Run(ctx, spec) -> (Result, error)

- internal/worktree/types.go

Workspace
- RepoRoot string
- Path string
- ID string

------------------------------------------------------------
3) Files: Exact files to create, modify, or delete

Create:
garl/
  cmd/garl/
    main.go
  internal/
    cli/
      flags.go
      run.go
    worktree/
      worktree.go
      types.go
    workflow/
      load.go
      model.go
      select.go
      plan.go
    runtime/
      defaults.go
      interpolate.go
      state.go
    envfiles/
      envfiles.go
      parse.go
      types.go
    executor/
      executor.go
      host.go
    util/
      git.go
      env.go
      fs.go
  go.mod
  go.sum
  README.md

Modify:
- None (initially)

Delete:
- None

------------------------------------------------------------
4) Functions: New and modified functions with signatures

CLI
- internal/cli/run.go
  - RunCommand(args []string) int

Worktree
- internal/worktree/worktree.go
  - CreateWorkspace(ctx context.Context, repoRoot string) (*Workspace, error)
  - RemoveWorkspace(ctx context.Context, ws *Workspace) error

Git helpers
- internal/util/git.go
  - FindGitRoot(ctx context.Context, startDir string) (string, error)
  - GitHeadSHA(ctx context.Context, repoRoot string) (string, error)
  - GitHeadRef(ctx context.Context, repoRoot string) (string, error)  (best-effort refs/heads/main etc.)

Workflow load/selection
- internal/workflow/load.go
  - LoadWorkflowFile(path string) (*WorkflowFile, error)

- internal/workflow/select.go
  - DiscoverWorkflowFiles(repoRoot string) ([]string, error)
  - SelectWorkflow(paths []string, workflowPath string) (string, error)
  - SelectJob(wf *WorkflowFile, jobID string) (string, Job, error)

Planning (resolve env/defaults/working dir/shell)
- internal/workflow/plan.go
  - BuildJobPlan(repoRoot, workflowPath, jobID string, wf *WorkflowFile, job Job) (*JobPlan, error)

Runtime defaults + env layering
- internal/runtime/defaults.go
  - DefaultBaseEnv(repoRoot, workspacePath string, sha string, ref string) map[string]string
  - MergeEnv(maps ...map[string]string) map[string]string

- internal/runtime/interpolate.go
  - InterpolateExpressions(s string, env map[string]string) string
    (MVP: support only ${{ env.X }} and ${{ github.workspace }} if implemented early)

Environment files
- internal/envfiles/envfiles.go
  - NewStepFiles(baseDir string) (*StepFiles, error)
  - CleanupStepFiles(sf *StepFiles) error

- internal/envfiles/parse.go
  - ApplyGITHUB_ENV(state map[string]string, envFilePath string) error
  - ApplyGITHUB_PATH(pathPrepend *[]string, pathFilePath string) error
  - ReadGITHUB_OUTPUT(outputs map[string]string, outputFilePath string) error

Execution
- internal/executor/host.go
  - type HostExecutor struct{}
  - (h HostExecutor) Run(ctx context.Context, spec ExecSpec) (Result, error)

- internal/cli/run.go (core loop)
  - RunJobPlan(ctx context.Context, ws *worktree.Workspace, plan *workflow.JobPlan) error

------------------------------------------------------------
5) Classes: Class modifications and inheritance details

Note: Go has no inheritance; use interfaces + structs.

Executor abstraction
- executor.Executor interface
- executor.HostExecutor concrete implementation
- Future: executor.DockerExecutor implementing the same interface (no changes to plan/execution loop required)

Runtime state container
- runtime.State struct holds:
  - persistent env map for subsequent steps
  - step outputs (if implemented)
  - PATH prepend list

Workspace lifecycle
- worktree.Workspace struct tracks repoRoot/path/id
- Functions manage lifecycle; avoid global state

------------------------------------------------------------
6) Dependencies: Package requirements and versions

Required (recommended)
- Go toolchain: Go 1.22+

External modules
- YAML parsing: gopkg.in/yaml.v3 v3.0.1

Optional (defer for MVP)
- CLI framework: github.com/spf13/cobra
- Logging: stdlib log/slog (or zap)

MVP can be stdlib + yaml.v3 + git CLI only.

------------------------------------------------------------
7) Testing: Validation strategies and test requirements

Unit tests
1. Env merge precedence
   - workflow env < job env < step env
2. Environment file parsers
   - GITHUB_ENV: KEY=VALUE and multiline delimiter format
   - GITHUB_PATH: multiple lines
   - GITHUB_OUTPUT: name=value and delimiter format (if implemented)
3. Expression interpolation (if included)
   - ${{ env.X }} and ${{ github.workspace }} behavior

Integration tests
1. Worktree isolation
   - Create workspace, mutate files, ensure repo root unaffected after cleanup
2. Run step execution
   - step1 writes to GITHUB_ENV, step2 reads it
3. PATH augmentation
   - step1 appends to GITHUB_PATH, step2 resolves a command from that directory

Golden tests (recommended)
- Keep workflow YAML fixtures under testdata/ and compare produced JobPlan snapshots

------------------------------------------------------------
8) Implementation Order: Step-by-step execution sequence

1. Scaffold CLI + repo detection
   - garl run with --workflow and --job
   - FindGitRoot
2. Worktree workspace lifecycle
   - CreateWorkspace / RemoveWorkspace
3. Workflow discovery + YAML loader
   - discover .github/workflows/*.yml
   - load YAML into WorkflowFile
4. Job selection + step planning
   - select job, flatten steps
   - resolve per-step shell and working-directory
5. HostExecutor for run steps
   - execute run with correct Dir and env
   - handle exit codes and errors clearly
6. Env layering
   - implement MergeEnv and apply in plan
7. Environment files (highest leverage)
   - create per-step temp dir
   - set GITHUB_ENV, GITHUB_PATH, GITHUB_OUTPUT, GITHUB_STEP_SUMMARY
   - after step: parse/apply to runtime state for next step
8. Minimal default env
   - GITHUB_WORKSPACE, GITHUB_ACTIONS, CI, GITHUB_SHA, GITHUB_REF
9. Optional early win: minimal ${{ }} interpolation
   - keep limited and non-Turing-complete
10. Testing + hardening
   - unit tests for envfiles and env merge
   - integration tests for worktree + step chaining
