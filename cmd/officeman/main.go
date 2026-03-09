package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/github-flaboy/officeman/internal/api"
	"github.com/github-flaboy/officeman/internal/app"
	"github.com/github-flaboy/officeman/internal/buildinfo"
	"github.com/github-flaboy/officeman/internal/config"
	"github.com/github-flaboy/officeman/internal/document"
	"github.com/github-flaboy/officeman/internal/excel"
	"github.com/github-flaboy/officeman/internal/httpserver"
	"github.com/github-flaboy/officeman/internal/storage"
	"github.com/github-flaboy/officeman/internal/vfs"
)

func main() {
	cfg := config.Load()
	addr := fmt.Sprintf(":%d", cfg.Port)
	svc := app.WorkbookService{
		Resolver: resolverFunc(vfs.ResolveFile),
		Store:    storage.NewS3Store(nil),
		Engine:   excel.NewEngine(),
	}
	docSvc := app.DocumentService{
		Resolver: resolverFunc(vfs.ResolveFile),
		Store:    storage.NewS3Store(nil),
		Engine:   document.NewEngine(),
	}
	srv := &http.Server{
		Addr:    addr,
		Handler: httpserver.NewHandler(svc, docSvc),
	}
	log.Printf("officeman starting version=%s addr=%s", buildinfo.Version, addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}

type resolverFunc func(api.VFSContext, string) (vfs.ResolvedFile, *vfs.ResolveError)

func (f resolverFunc) ResolveFile(ctx api.VFSContext, filePath string) (vfs.ResolvedFile, *vfs.ResolveError) {
	return f(ctx, filePath)
}
