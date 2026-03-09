# Officeman XLSX VFS Service Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build an independent Go service in `officeman` that accepts session-scoped VFS config plus a VFS target file path, resolves that path to S3, and performs structured `.xlsx` operations against the resolved object.

**Architecture:** The service follows the `browserd` API style: resource-oriented routes, one typed request struct per action, and a shared `{data,error}` response envelope. Each workbook action request carries `vfs.mounts`, `vfs.s3_sets`, `vfs.template_vars`, and `filePath`; the service resolves that VFS path with botrunner-compatible semantics, loads or creates an XLSX workbook via Excelize, applies the requested action, then writes the resulting bytes back to the same S3 object when the action is mutating.

**Tech Stack:** Go 1.24, standard library `net/http`, `github.com/xuri/excelize/v2`, `github.com/aws/aws-sdk-go-v2`, `github.com/aws/aws-sdk-go-v2/service/s3`

---

## Assumptions

- Target directory `/Users/wanglei/Projects/github-flaboy/officeman` is currently empty and not a git repository.
- V1 only supports `.xlsx`.
- V1 only supports these routes/actions: `create`, `meta`, `write-cells`, `append-rows`, `add-sheet`, `rename-sheet`, `delete-sheet`.
- External callers only provide VFS paths. They never provide raw S3 bucket/key.
- Configuration naming should stay as close as practical to `browserd`: keep S3 field names `endpoint`, `region`, `access_key_id`, `secret_access_key`, `force_path_style`, and use `OFFICEMAN_*` env names that mirror `BROWSERD_*` suffixes.
- VFS resolve behavior must stay aligned with current botrunner logic:
  - virtual path must start with `/`
  - reject `..`
  - use longest mount prefix match
  - `relativePath = filePath - mountPath`
  - template-substitute mount path and normalize to trailing slash
  - `s3Key = resolvedPrefix + relativePath`

### Task 1: Initialize Repository Skeleton

**Files:**
- Create: `go.mod`
- Create: `.gitignore`
- Create: `README.md`
- Create: `cmd/officeman/main.go`
- Create: `internal/buildinfo/buildinfo.go`
- Create: `docs/plans/2026-03-09-officeman-xlsx-vfs-service.md`

**Step 1: Create module and ignore files**
```go
module github.com/github-flaboy/officeman

go 1.24
```

```gitignore
bin/
dist/
.DS_Store
coverage.out
```

**Step 2: Add minimal boot entrypoint**
```go
package main

import "log"

func main() {
	log.Println("officeman starting")
}
```

**Step 3: Add minimal README with service scope**
```md
# officeman

Internal Go service for VFS-backed `.xlsx` operations.
```

**Step 4: Verify bootstrap builds**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./...`
Expected: PASS with no test files yet.

**Step 5: Commit**
```bash
cd /Users/wanglei/Projects/github-flaboy/officeman
git init
git add go.mod .gitignore README.md cmd/officeman/main.go internal/buildinfo/buildinfo.go docs/plans/2026-03-09-officeman-xlsx-vfs-service.md
git commit -m "chore: bootstrap officeman service"
```

### Task 2: Define Browserd-Style Typed HTTP Contracts

**Files:**
- Create: `internal/api/types.go`
- Create: `internal/api/types_test.go`

**Step 1: Write failing tests for request validation**
```go
func TestWorkbookBaseRequest_ValidateRejectsNonAbsolutePath(t *testing.T) {}

func TestWriteCellsRequest_ValidateRequiresSheetName(t *testing.T) {}

func TestCreateWorkbookRequest_ValidateAcceptsSheets(t *testing.T) {}
```

**Step 2: Run tests to verify they fail**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./internal/api -run 'TestWorkbookBaseRequest|TestWriteCellsRequest|TestCreateWorkbookRequest' -v`
Expected: FAIL because request types and validation do not exist.

**Step 3: Add typed request/response model**
```go
type WorkbookBaseRequest struct {
	RequestID string     `json:"requestId,omitempty"`
	VFS       VFSContext `json:"vfs"`
	FilePath  string     `json:"filePath"`
}
```

Add typed request structs for:

- `CreateWorkbookRequest`
- `WorkbookMetaRequest`
- `WriteCellsRequest`
- `AppendRowsRequest`
- `AddSheetRequest`
- `RenameSheetRequest`
- `DeleteSheetRequest`

Add `Validate()` methods on each request type. Keep request fields explicit. Do not use `operation + args` or `map[string]any`.

**Step 4: Run tests to verify they pass**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./internal/api -run 'TestWorkbookBaseRequest|TestWriteCellsRequest|TestCreateWorkbookRequest' -v`
Expected: PASS.

**Step 5: Commit**
```bash
cd /Users/wanglei/Projects/github-flaboy/officeman
git add internal/api/types.go internal/api/types_test.go
git commit -m "feat: add workbook api contracts"
```

### Task 3: Implement VFS Resolve Layer with Botrunner-Compatible Semantics

**Files:**
- Create: `internal/vfs/types.go`
- Create: `internal/vfs/resolve.go`
- Create: `internal/vfs/resolve_test.go`

**Step 1: Write failing tests for VFS resolve**
```go
func TestResolvePath_UsesLongestMountPrefix(t *testing.T) {}

func TestResolvePath_RejectsTraversal(t *testing.T) {}

func TestResolvePath_RendersTemplateVarsIntoS3Key(t *testing.T) {}

func TestResolvePath_RejectsUnknownBucketAlias(t *testing.T) {}
```

**Step 2: Run tests to verify they fail**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./internal/vfs -run TestResolvePath -v`
Expected: FAIL because resolver is missing.

**Step 3: Implement resolver**
```go
type ResolvedFile struct {
	MountPath   string
	Permission  string
	BucketAlias string
	BucketName  string
	S3Prefix    string
	RelativePath string
	S3Key       string
}
```

Implement:

- `ValidateVirtualPath(path string) error`
- `ResolveFile(ctx VFSContext, filePath string) (ResolvedFile, error)`
- `normalizeS3Prefix`
- `resolveTemplate`
- longest-prefix mount selection

Map errors to stable codes:

- `VFS_INVALID_PATH`
- `VFS_PATH_TRAVERSAL`
- `VFS_PATH_NOT_MOUNTED`
- `VFS_MISSING_TEMPLATE_VAR`
- `VFS_S3_SET_NOT_FOUND`

**Step 4: Run tests to verify they pass**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./internal/vfs -run TestResolvePath -v`
Expected: PASS.

**Step 5: Commit**
```bash
cd /Users/wanglei/Projects/github-flaboy/officeman
git add internal/vfs/types.go internal/vfs/resolve.go internal/vfs/resolve_test.go
git commit -m "feat: add vfs path resolver"
```

### Task 4: Implement S3 Storage Adapter

**Files:**
- Create: `internal/storage/s3store.go`
- Create: `internal/storage/s3store_test.go`

**Step 1: Write failing tests for object read/write**
```go
func TestS3Store_GetObjectBytes(t *testing.T) {}

func TestS3Store_PutObjectBytes(t *testing.T) {}
```

**Step 2: Run tests to verify they fail**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./internal/storage -run TestS3Store -v`
Expected: FAIL because store implementation is missing.

**Step 3: Implement minimal adapter over AWS SDK v2**
```go
type ObjectStore interface {
	GetObjectBytes(ctx context.Context, cfg S3SetConfig, key string) ([]byte, error)
	PutObjectBytes(ctx context.Context, cfg S3SetConfig, key string, body []byte, contentType string) error
	HeadObject(ctx context.Context, cfg S3SetConfig, key string) (bool, error)
}
```

Implementation details:

- create S3 client from request-scoped `S3SetConfig`
- support custom `endpoint`
- support `force_path_style`
- keep config field names aligned with `browserd/internal/config/config.go` where practical
- use `PutObject` for final workbook upload

**Step 4: Run tests to verify they pass**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./internal/storage -run TestS3Store -v`
Expected: PASS using mocked client interface.

**Step 5: Commit**
```bash
cd /Users/wanglei/Projects/github-flaboy/officeman
git add internal/storage/s3store.go internal/storage/s3store_test.go
git commit -m "feat: add s3 storage adapter"
```

### Task 5: Implement XLSX Engine

**Files:**
- Create: `internal/excel/engine.go`
- Create: `internal/excel/engine_test.go`

**Step 1: Write failing tests for supported workbook operations**
```go
func TestEngine_CreateWorkbook(t *testing.T) {}

func TestEngine_WriteCells(t *testing.T) {}

func TestEngine_AppendRows(t *testing.T) {}

func TestEngine_RenameDeleteAndAddSheet(t *testing.T) {}

func TestEngine_ReadWorkbookMeta(t *testing.T) {}
```

**Step 2: Run tests to verify they fail**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./internal/excel -run TestEngine -v`
Expected: FAIL because engine is missing.

**Step 3: Implement engine with Excelize**
```go
type Engine interface {
	CreateWorkbook(req api.CreateWorkbookRequest) ([]byte, WorkbookMeta, error)
	ReadWorkbookMeta(src []byte) (WorkbookMeta, error)
	WriteCells(src []byte, req api.WriteCellsRequest) ([]byte, WorkbookMeta, error)
	AppendRows(src []byte, req api.AppendRowsRequest) ([]byte, WorkbookMeta, error)
	AddSheet(src []byte, req api.AddSheetRequest) ([]byte, WorkbookMeta, error)
	RenameSheet(src []byte, req api.RenameSheetRequest) ([]byte, WorkbookMeta, error)
	DeleteSheet(src []byte, req api.DeleteSheetRequest) ([]byte, WorkbookMeta, error)
}
```

Rules:

- only `.xlsx`
- `create` creates a workbook with requested sheets
- `write-cells` writes a 2D value matrix from `startCell`
- `append-rows` appends after the last used row
- `add-sheet`, `rename-sheet`, `delete-sheet` operate by exact sheet name
- `meta` returns sheet names and dimensions only

**Step 4: Run tests to verify they pass**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./internal/excel -run TestEngine -v`
Expected: PASS.

**Step 5: Commit**
```bash
cd /Users/wanglei/Projects/github-flaboy/officeman
git add internal/excel/engine.go internal/excel/engine_test.go go.mod go.sum
git commit -m "feat: add xlsx engine"
```

### Task 6: Implement Workbook Service Use Case

**Files:**
- Create: `internal/app/workbook_service.go`
- Create: `internal/app/workbook_service_test.go`

**Step 1: Write failing application tests**
```go
func TestWorkbookService_CreateWorkbookToResolvedS3Key(t *testing.T) {}

func TestWorkbookService_RejectsWriteOnReadOnlyMount(t *testing.T) {}

func TestWorkbookService_ReadWorkbookMetaDoesNotWriteBack(t *testing.T) {}
```

**Step 2: Run tests to verify they fail**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./internal/app -run TestWorkbookService -v`
Expected: FAIL because workbook service is missing.

**Step 3: Implement orchestration**
```go
type WorkbookService struct {
	Resolver vfs.Resolver
	Store    storage.ObjectStore
	Engine   excel.Engine
}
```

Flow:

1. validate typed request
2. resolve `filePath`
3. reject mutating actions on `read_only`
4. for mutating actions:
   - read existing object when needed
   - create or mutate workbook
   - write bytes back to resolved S3 key
5. for `meta`:
   - read existing object
   - return workbook metadata without write-back

**Step 4: Run tests to verify they pass**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./internal/app -run TestWorkbookService -v`
Expected: PASS.

**Step 5: Commit**
```bash
cd /Users/wanglei/Projects/github-flaboy/officeman
git add internal/app/workbook_service.go internal/app/workbook_service_test.go
git commit -m "feat: add workbook service use case"
```

### Task 7: Expose Browserd-Style HTTP Routes and Error Envelope

**Files:**
- Create: `internal/httpserver/server.go`
- Create: `internal/httpserver/server_test.go`
- Modify: `cmd/officeman/main.go`

**Step 1: Write failing HTTP tests**
```go
func TestServer_CreateReturns200(t *testing.T) {}

func TestServer_WriteCellsReturns400ForBadRequest(t *testing.T) {}

func TestServer_RenameSheetReturns409ForBusinessConflict(t *testing.T) {}
```

**Step 2: Run tests to verify they fail**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./internal/httpserver -run TestServer -v`
Expected: FAIL because HTTP server is missing.

**Step 3: Implement server**
```go
func NewHandler(svc *app.WorkbookService) http.Handler
```

Behavior:

- copy the shared `{data,error}` envelope pattern from `browserd/internal/types/api.go`
- expose:
  - `POST /v1/workbooks/create`
  - `POST /v1/workbooks/meta`
  - `POST /v1/workbooks/write-cells`
  - `POST /v1/workbooks/append-rows`
  - `POST /v1/workbooks/add-sheet`
  - `POST /v1/workbooks/rename-sheet`
  - `POST /v1/workbooks/delete-sheet`
- decode typed JSON request per route
- call the matching workbook service method
- return JSON envelope with `data`, `error`
- map validation errors to `400`
- map permission/conflict/not-found style business errors to `403/404/409`
- map unexpected errors to `500`

**Step 4: Run tests to verify they pass**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./internal/httpserver -run TestServer -v`
Expected: PASS.

**Step 5: Commit**
```bash
cd /Users/wanglei/Projects/github-flaboy/officeman
git add internal/httpserver/server.go internal/httpserver/server_test.go cmd/officeman/main.go
git commit -m "feat: expose workbook http routes"
```

### Task 8: Add End-to-End Local Integration Test

**Files:**
- Create: `internal/e2e/execute_local_test.go`
- Modify: `README.md`

**Step 1: Write failing integration test**
```go
func TestExecuteFlow_CreateThenWriteCells(t *testing.T) {}
```

Test shape:

- construct in-memory VFS config
- mount `/workdir/` to a fake S3 key prefix
- call service twice:
  - `create`
  - `write-cells`
- assert final uploaded bytes can be reopened as valid `.xlsx`
- assert resolved S3 key equals expected workspace path

**Step 2: Run test to verify it fails**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./internal/e2e -run TestExecuteFlow_CreateThenWriteCells -v`
Expected: FAIL because test harness is incomplete.

**Step 3: Implement integration harness and README usage**
```md
## API

POST /v1/workbooks/write-cells
```

Document:

- shared envelope shape
- route list and request shape
- supported actions
- VFS path to S3 key mapping example
- current non-goals

**Step 4: Run full test suite**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./...`
Expected: PASS.

**Step 5: Commit**
```bash
cd /Users/wanglei/Projects/github-flaboy/officeman
git add internal/e2e/execute_local_test.go README.md
git commit -m "test: add end-to-end execute flow"
```

### Task 9: Final Checklist Review

**Files:**
- Modify: `README.md` if review finds gaps
- Modify: `docs/plans/2026-03-09-officeman-xlsx-vfs-service.md` only if implementation changed scope

**Step 1: Re-check plan items against implementation**
Review checklist:

- all routes and actions match plan scope
- VFS resolve semantics match botrunner behavior
- no raw S3 key exposed in external API
- `.xlsx` only
- read-only mounts cannot mutate
- tests cover mapping and workbook mutations

**Step 2: Run final verification commands**
Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./...`
Expected: PASS.

Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && gofmt -w ./cmd ./internal`
Expected: formatting applied with no errors.

Run: `cd /Users/wanglei/Projects/github-flaboy/officeman && go test ./...`
Expected: PASS again after formatting.

**Step 3: Fix any review drift minimally**
```go
// Keep changes minimal: only align code/docs/tests with the approved plan.
```

**Step 4: Prepare final change summary**
Capture:

- supported actions
- VFS mapping guarantee
- known non-goals

**Step 5: Commit**
```bash
cd /Users/wanglei/Projects/github-flaboy/officeman
git add README.md docs/plans/2026-03-09-officeman-xlsx-vfs-service.md .
git commit -m "chore: finalize officeman xlsx vfs service"
```
