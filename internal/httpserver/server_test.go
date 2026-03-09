package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/github-flaboy/officeman/internal/api"
	"github.com/github-flaboy/officeman/internal/app"
	"github.com/github-flaboy/officeman/internal/excel"
	"github.com/github-flaboy/officeman/internal/vfs"
)

type fakeWorkbookService struct {
	createFn      func(context.Context, api.CreateWorkbookRequest) (app.Result, *app.ServiceError)
	metaFn        func(context.Context, api.WorkbookMetaRequest) (app.Result, *app.ServiceError)
	writeCellsFn  func(context.Context, api.WriteCellsRequest) (app.Result, *app.ServiceError)
	appendRowsFn  func(context.Context, api.AppendRowsRequest) (app.Result, *app.ServiceError)
	addSheetFn    func(context.Context, api.AddSheetRequest) (app.Result, *app.ServiceError)
	renameSheetFn func(context.Context, api.RenameSheetRequest) (app.Result, *app.ServiceError)
	deleteSheetFn func(context.Context, api.DeleteSheetRequest) (app.Result, *app.ServiceError)
}

func (f fakeWorkbookService) Create(ctx context.Context, req api.CreateWorkbookRequest) (app.Result, *app.ServiceError) {
	return f.createFn(ctx, req)
}
func (f fakeWorkbookService) Meta(ctx context.Context, req api.WorkbookMetaRequest) (app.Result, *app.ServiceError) {
	return f.metaFn(ctx, req)
}
func (f fakeWorkbookService) WriteCells(ctx context.Context, req api.WriteCellsRequest) (app.Result, *app.ServiceError) {
	return f.writeCellsFn(ctx, req)
}
func (f fakeWorkbookService) AppendRows(ctx context.Context, req api.AppendRowsRequest) (app.Result, *app.ServiceError) {
	return f.appendRowsFn(ctx, req)
}
func (f fakeWorkbookService) AddSheet(ctx context.Context, req api.AddSheetRequest) (app.Result, *app.ServiceError) {
	return f.addSheetFn(ctx, req)
}
func (f fakeWorkbookService) RenameSheet(ctx context.Context, req api.RenameSheetRequest) (app.Result, *app.ServiceError) {
	return f.renameSheetFn(ctx, req)
}
func (f fakeWorkbookService) DeleteSheet(ctx context.Context, req api.DeleteSheetRequest) (app.Result, *app.ServiceError) {
	return f.deleteSheetFn(ctx, req)
}

type envelope struct {
	Data  map[string]any `json:"data"`
	Error map[string]any `json:"error"`
}

func TestServer_CreateReturns200(t *testing.T) {
	handler := NewHandler(fakeWorkbookService{
		createFn: func(_ context.Context, _ api.CreateWorkbookRequest) (app.Result, *app.ServiceError) {
			return app.Result{
				Resolved: vfs.ResolvedFile{S3Key: "tenants/t1/teams/team1/cases/c1/workspace/report.xlsx"},
				Meta:     excel.WorkbookMeta{Sheets: []excel.SheetMeta{{Name: "Sheet1"}}},
			}, nil
		},
		metaFn:        noopMeta,
		writeCellsFn:  noopWriteCells,
		appendRowsFn:  noopAppendRows,
		addSheetFn:    noopAddSheet,
		renameSheetFn: noopRenameSheet,
		deleteSheetFn: noopDeleteSheet,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/workbooks/create", bytes.NewBufferString(validCreateBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rr.Code)
	}
	var out envelope
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out.Error != nil {
		t.Fatalf("error = %+v, want nil", out.Error)
	}
}

func TestServer_WriteCellsReturns400ForBadRequest(t *testing.T) {
	handler := NewHandler(fakeWorkbookService{
		createFn:      noopCreate,
		metaFn:        noopMeta,
		writeCellsFn:  noopWriteCells,
		appendRowsFn:  noopAppendRows,
		addSheetFn:    noopAddSheet,
		renameSheetFn: noopRenameSheet,
		deleteSheetFn: noopDeleteSheet,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/workbooks/write-cells", bytes.NewBufferString(`{"filePath":"/workdir/report.xlsx"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestServer_RenameSheetReturns409ForBusinessConflict(t *testing.T) {
	handler := NewHandler(fakeWorkbookService{
		createFn:     noopCreate,
		metaFn:       noopMeta,
		writeCellsFn: noopWriteCells,
		appendRowsFn: noopAppendRows,
		addSheetFn:   noopAddSheet,
		renameSheetFn: func(_ context.Context, _ api.RenameSheetRequest) (app.Result, *app.ServiceError) {
			return app.Result{}, &app.ServiceError{Code: "SHEET_ALREADY_EXISTS", Message: "sheet already exists"}
		},
		deleteSheetFn: noopDeleteSheet,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/workbooks/rename-sheet", bytes.NewBufferString(validRenameBody))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", rr.Code)
	}
}

var validCreateBody = `{
  "vfs": {
    "mounts": {
      "/workdir/": {
        "permission": "read_write",
        "bucket": "private",
        "path": "tenants/{tenant_id}/teams/{team_id}/cases/{case_id}/workspace/",
        "ttl_ms": 30000
      }
    },
    "s3_sets": {
      "private": { "bucket": "private-bucket" }
    },
    "template_vars": {
      "tenant_id": "t1",
      "team_id": "team1",
      "case_id": "c1"
    }
  },
  "filePath": "/workdir/report.xlsx",
  "sheets": [{ "name": "Sheet1" }]
}`

var validRenameBody = `{
  "vfs": {
    "mounts": {
      "/workdir/": {
        "permission": "read_write",
        "bucket": "private",
        "path": "tenants/{tenant_id}/teams/{team_id}/cases/{case_id}/workspace/",
        "ttl_ms": 30000
      }
    },
    "s3_sets": {
      "private": { "bucket": "private-bucket" }
    },
    "template_vars": {
      "tenant_id": "t1",
      "team_id": "team1",
      "case_id": "c1"
    }
  },
  "filePath": "/workdir/report.xlsx",
  "fromSheetName": "Input",
  "toSheetName": "Summary"
}`

func noopCreate(context.Context, api.CreateWorkbookRequest) (app.Result, *app.ServiceError) {
	return app.Result{}, nil
}
func noopMeta(context.Context, api.WorkbookMetaRequest) (app.Result, *app.ServiceError) {
	return app.Result{}, nil
}
func noopWriteCells(context.Context, api.WriteCellsRequest) (app.Result, *app.ServiceError) {
	return app.Result{}, nil
}
func noopAppendRows(context.Context, api.AppendRowsRequest) (app.Result, *app.ServiceError) {
	return app.Result{}, nil
}
func noopAddSheet(context.Context, api.AddSheetRequest) (app.Result, *app.ServiceError) {
	return app.Result{}, nil
}
func noopRenameSheet(context.Context, api.RenameSheetRequest) (app.Result, *app.ServiceError) {
	return app.Result{}, nil
}
func noopDeleteSheet(context.Context, api.DeleteSheetRequest) (app.Result, *app.ServiceError) {
	return app.Result{}, nil
}
