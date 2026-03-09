package app

import (
	"context"

	"github.com/github-flaboy/officeman/internal/api"
	"github.com/github-flaboy/officeman/internal/document"
	"github.com/github-flaboy/officeman/internal/storage"
	"github.com/github-flaboy/officeman/internal/vfs"
)

type DocumentService struct {
	Resolver resolver
	Store    storage.ObjectStore
	Engine   document.Engine
}

type DocumentResult struct {
	Resolved vfs.ResolvedFile `json:"resolved"`
	Meta     document.Meta    `json:"meta"`
}

func (s DocumentService) Write(ctx context.Context, req api.WriteDocumentRequest) (DocumentResult, *ServiceError) {
	if err := req.Validate(); err != nil {
		return DocumentResult{}, invalid(err.Message)
	}
	resolved, cfg, err := s.resolveConfig(req.VFS, req.FilePath)
	if err != nil {
		return DocumentResult{}, err
	}
	if resolved.Permission != "read_write" {
		return DocumentResult{}, permissionDenied("mount does not allow mutation")
	}
	body, meta, writeErr := s.Engine.Write(req)
	if writeErr != nil {
		return DocumentResult{}, internalErr(writeErr.Error())
	}
	if putErr := s.Store.PutObjectBytes(ctx, cfg, resolved.S3Key, body, storage.DOCXContentType); putErr != nil {
		return DocumentResult{}, internalErr(putErr.Error())
	}
	return DocumentResult{Resolved: resolved, Meta: meta}, nil
}

func (s DocumentService) Read(ctx context.Context, req api.ReadDocumentRequest) (DocumentResult, *ServiceError) {
	if err := req.Validate(); err != nil {
		return DocumentResult{}, invalid(err.Message)
	}
	resolved, cfg, err := s.resolveConfig(req.VFS, req.FilePath)
	if err != nil {
		return DocumentResult{}, err
	}
	body, getErr := s.Store.GetObjectBytes(ctx, cfg, resolved.S3Key)
	if getErr != nil {
		return DocumentResult{}, internalErr(getErr.Error())
	}
	meta, readErr := s.Engine.Read(body)
	if readErr != nil {
		return DocumentResult{}, internalErr(readErr.Error())
	}
	return DocumentResult{Resolved: resolved, Meta: meta}, nil
}

func (s DocumentService) resolveConfig(vfsCtx api.VFSContext, filePath string) (vfs.ResolvedFile, api.S3SetConfig, *ServiceError) {
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
