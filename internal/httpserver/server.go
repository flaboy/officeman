package httpserver

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/github-flaboy/officeman/internal/api"
	"github.com/github-flaboy/officeman/internal/app"
	apitypes "github.com/github-flaboy/officeman/internal/types"
)

type workbookService interface {
	Create(ctx context.Context, req api.CreateWorkbookRequest) (app.Result, *app.ServiceError)
	Meta(ctx context.Context, req api.WorkbookMetaRequest) (app.Result, *app.ServiceError)
	WriteCells(ctx context.Context, req api.WriteCellsRequest) (app.Result, *app.ServiceError)
	AppendRows(ctx context.Context, req api.AppendRowsRequest) (app.Result, *app.ServiceError)
	AddSheet(ctx context.Context, req api.AddSheetRequest) (app.Result, *app.ServiceError)
	RenameSheet(ctx context.Context, req api.RenameSheetRequest) (app.Result, *app.ServiceError)
	DeleteSheet(ctx context.Context, req api.DeleteSheetRequest) (app.Result, *app.ServiceError)
}

type documentService interface {
	Write(ctx context.Context, req api.WriteDocumentRequest) (app.DocumentResult, *app.ServiceError)
	Read(ctx context.Context, req api.ReadDocumentRequest) (app.DocumentResult, *app.ServiceError)
}

func NewHandler(workbooks workbookService, documents documentService) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/healthz":
			apitypes.WriteOK(w, http.StatusOK, map[string]any{"ok": true})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/workbooks/create":
			handle(w, r, workbooks.Create)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/workbooks/meta":
			handle(w, r, workbooks.Meta)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/workbooks/write-cells":
			handle(w, r, workbooks.WriteCells)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/workbooks/append-rows":
			handle(w, r, workbooks.AppendRows)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/workbooks/add-sheet":
			handle(w, r, workbooks.AddSheet)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/workbooks/rename-sheet":
			handle(w, r, workbooks.RenameSheet)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/workbooks/delete-sheet":
			handle(w, r, workbooks.DeleteSheet)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/documents/write":
			handle(w, r, documents.Write)
		case r.Method == http.MethodPost && r.URL.Path == "/v1/documents/read":
			handle(w, r, documents.Read)
		default:
			apitypes.WriteErr(w, http.StatusNotFound, "NOT_FOUND", "route not found")
		}
	})
}

type validator interface {
	Validate() *api.ValidationError
}

func handle[T validator, R any](w http.ResponseWriter, r *http.Request, fn func(context.Context, T) (R, *app.ServiceError)) {
	var req T
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		apitypes.WriteErr(w, http.StatusBadRequest, "INVALID_REQUEST", "invalid json body")
		return
	}
	if err := req.Validate(); err != nil {
		apitypes.WriteErr(w, http.StatusBadRequest, err.Code, err.Message)
		return
	}
	out, svcErr := fn(r.Context(), req)
	if svcErr != nil {
		apitypes.WriteErr(w, statusCodeFor(svcErr.Code), svcErr.Code, svcErr.Message)
		return
	}
	apitypes.WriteOK(w, http.StatusOK, out)
}

func statusCodeFor(code string) int {
	switch code {
	case "INVALID_REQUEST", "VFS_INVALID_PATH", "VFS_PATH_TRAVERSAL", "VFS_PATH_NOT_MOUNTED", "VFS_MISSING_TEMPLATE_VAR", "VFS_S3_SET_NOT_FOUND":
		return http.StatusBadRequest
	case "VFS_PERMISSION_DENIED":
		return http.StatusForbidden
	case "FILE_NOT_FOUND":
		return http.StatusNotFound
	case "WORKBOOK_ALREADY_EXISTS", "SHEET_ALREADY_EXISTS":
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
