package app

import (
	"context"

	"github.com/github-flaboy/officeman/internal/api"
	"github.com/github-flaboy/officeman/internal/excel"
	"github.com/github-flaboy/officeman/internal/storage"
	"github.com/github-flaboy/officeman/internal/vfs"
)

type WorkbookService struct {
	Resolver resolver
	Store    storage.ObjectStore
	Engine   excel.Engine
}

type resolver interface {
	ResolveFile(ctx api.VFSContext, filePath string) (vfs.ResolvedFile, *vfs.ResolveError)
}

type Result struct {
	Resolved vfs.ResolvedFile   `json:"resolved"`
	Meta     excel.WorkbookMeta `json:"meta"`
}

type ServiceError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *ServiceError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (s WorkbookService) Create(ctx context.Context, req api.CreateWorkbookRequest) (Result, *ServiceError) {
	if err := req.Validate(); err != nil {
		return Result{}, invalid(err.Message)
	}
	resolved, cfg, err := s.resolveConfig(req.VFS, req.FilePath)
	if err != nil {
		return Result{}, err
	}
	exists, headErr := s.Store.HeadObject(ctx, cfg, resolved.S3Key)
	if headErr != nil {
		return Result{}, internalErr(headErr.Error())
	}
	if exists {
		return Result{}, conflict("target workbook already exists")
	}

	body, meta, createErr := s.Engine.CreateWorkbook(req)
	if createErr != nil {
		return Result{}, internalErr(createErr.Error())
	}
	if putErr := s.Store.PutObjectBytes(ctx, cfg, resolved.S3Key, body, storage.XLSXContentType); putErr != nil {
		return Result{}, internalErr(putErr.Error())
	}
	return Result{Resolved: resolved, Meta: meta}, nil
}

func (s WorkbookService) Meta(ctx context.Context, req api.WorkbookMetaRequest) (Result, *ServiceError) {
	if err := req.Validate(); err != nil {
		return Result{}, invalid(err.Message)
	}
	resolved, cfg, err := s.resolveConfig(req.VFS, req.FilePath)
	if err != nil {
		return Result{}, err
	}
	body, getErr := s.Store.GetObjectBytes(ctx, cfg, resolved.S3Key)
	if getErr != nil {
		return Result{}, internalErr(getErr.Error())
	}
	meta, readErr := s.Engine.ReadWorkbookMeta(body)
	if readErr != nil {
		return Result{}, internalErr(readErr.Error())
	}
	return Result{Resolved: resolved, Meta: meta}, nil
}

func (s WorkbookService) WriteCells(ctx context.Context, req api.WriteCellsRequest) (Result, *ServiceError) {
	return s.mutate(ctx, req.WorkbookBaseRequest, func(body []byte) ([]byte, excel.WorkbookMeta, error) {
		return s.Engine.WriteCells(body, req)
	})
}

func (s WorkbookService) AppendRows(ctx context.Context, req api.AppendRowsRequest) (Result, *ServiceError) {
	return s.mutate(ctx, req.WorkbookBaseRequest, func(body []byte) ([]byte, excel.WorkbookMeta, error) {
		return s.Engine.AppendRows(body, req)
	})
}

func (s WorkbookService) AddSheet(ctx context.Context, req api.AddSheetRequest) (Result, *ServiceError) {
	return s.mutate(ctx, req.WorkbookBaseRequest, func(body []byte) ([]byte, excel.WorkbookMeta, error) {
		return s.Engine.AddSheet(body, req)
	})
}

func (s WorkbookService) RenameSheet(ctx context.Context, req api.RenameSheetRequest) (Result, *ServiceError) {
	return s.mutate(ctx, req.WorkbookBaseRequest, func(body []byte) ([]byte, excel.WorkbookMeta, error) {
		return s.Engine.RenameSheet(body, req)
	})
}

func (s WorkbookService) DeleteSheet(ctx context.Context, req api.DeleteSheetRequest) (Result, *ServiceError) {
	return s.mutate(ctx, req.WorkbookBaseRequest, func(body []byte) ([]byte, excel.WorkbookMeta, error) {
		return s.Engine.DeleteSheet(body, req)
	})
}

func (s WorkbookService) mutate(
	ctx context.Context,
	base api.WorkbookBaseRequest,
	apply func(body []byte) ([]byte, excel.WorkbookMeta, error),
) (Result, *ServiceError) {
	if err := base.Validate(); err != nil {
		return Result{}, invalid(err.Message)
	}
	resolved, cfg, err := s.resolveConfig(base.VFS, base.FilePath)
	if err != nil {
		return Result{}, err
	}
	if resolved.Permission != "read_write" {
		return Result{}, permissionDenied("mount does not allow mutation")
	}
	body, getErr := s.Store.GetObjectBytes(ctx, cfg, resolved.S3Key)
	if getErr != nil {
		return Result{}, internalErr(getErr.Error())
	}
	next, meta, applyErr := apply(body)
	if applyErr != nil {
		return Result{}, internalErr(applyErr.Error())
	}
	if putErr := s.Store.PutObjectBytes(ctx, cfg, resolved.S3Key, next, storage.XLSXContentType); putErr != nil {
		return Result{}, internalErr(putErr.Error())
	}
	return Result{Resolved: resolved, Meta: meta}, nil
}

func (s WorkbookService) resolveConfig(vfsCtx api.VFSContext, filePath string) (vfs.ResolvedFile, api.S3SetConfig, *ServiceError) {
	resolved, resolveErr := s.Resolver.ResolveFile(vfsCtx, filePath)
	if resolveErr != nil {
		return vfs.ResolvedFile{}, api.S3SetConfig{}, &ServiceError{Code: resolveErr.Code, Message: resolveErr.Message}
	}
	cfg, ok := vfsCtx.S3Sets[resolved.BucketAlias]
	if !ok {
		return vfs.ResolvedFile{}, api.S3SetConfig{}, &ServiceError{Code: "VFS_S3_SET_NOT_FOUND", Message: "bucket alias is not configured"}
	}
	return resolved, cfg, nil
}

func invalid(message string) *ServiceError {
	return &ServiceError{Code: "INVALID_REQUEST", Message: message}
}

func conflict(message string) *ServiceError {
	return &ServiceError{Code: "WORKBOOK_ALREADY_EXISTS", Message: message}
}

func permissionDenied(message string) *ServiceError {
	return &ServiceError{Code: "VFS_PERMISSION_DENIED", Message: message}
}

func internalErr(message string) *ServiceError {
	return &ServiceError{Code: "INTERNAL_ERROR", Message: message}
}
