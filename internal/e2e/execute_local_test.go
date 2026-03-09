package e2e

import (
	"bytes"
	"context"
	"testing"

	"github.com/github-flaboy/officeman/internal/api"
	"github.com/github-flaboy/officeman/internal/app"
	"github.com/github-flaboy/officeman/internal/document"
	"github.com/github-flaboy/officeman/internal/excel"
	"github.com/github-flaboy/officeman/internal/storage"
	"github.com/github-flaboy/officeman/internal/vfs"
	"github.com/xuri/excelize/v2"
)

type memoryStore struct {
	objects map[string][]byte
}

func (m *memoryStore) GetObjectBytes(_ context.Context, cfg api.S3SetConfig, key string) ([]byte, error) {
	return append([]byte(nil), m.objects[cfg.Bucket+":"+key]...), nil
}

func (m *memoryStore) PutObjectBytes(_ context.Context, cfg api.S3SetConfig, key string, body []byte, _ string) error {
	if m.objects == nil {
		m.objects = map[string][]byte{}
	}
	m.objects[cfg.Bucket+":"+key] = append([]byte(nil), body...)
	return nil
}

func (m *memoryStore) HeadObject(_ context.Context, cfg api.S3SetConfig, key string) (bool, error) {
	_, ok := m.objects[cfg.Bucket+":"+key]
	return ok, nil
}

type resolverFunc func(api.VFSContext, string) (vfs.ResolvedFile, *vfs.ResolveError)

func (f resolverFunc) ResolveFile(ctx api.VFSContext, filePath string) (vfs.ResolvedFile, *vfs.ResolveError) {
	return f(ctx, filePath)
}

func TestExecuteFlow_CreateThenWriteCells(t *testing.T) {
	store := &memoryStore{}
	svc := app.WorkbookService{
		Resolver: resolverFunc(vfs.ResolveFile),
		Store:    store,
		Engine:   excel.NewEngine(),
	}

	base := api.WorkbookBaseRequest{
		VFS: api.VFSContext{
			Mounts: map[string]api.VFSMount{
				"/workdir/": {
					Permission: "read_write",
					Bucket:     "private",
					Path:       "tenants/{tenant_id}/teams/{team_id}/cases/{case_id}/workspace/",
					TTLMS:      30000,
				},
			},
			S3Sets: map[string]api.S3SetConfig{
				"private": {Bucket: "private-bucket"},
			},
			TemplateVars: map[string]string{
				"tenant_id": "t1",
				"team_id":   "team1",
				"case_id":   "c1",
			},
		},
		FilePath: "/workdir/report.xlsx",
	}

	createOut, err := svc.Create(context.Background(), api.CreateWorkbookRequest{
		WorkbookBaseRequest: base,
		Sheets:              []api.CreateWorkbookSheet{{Name: "Sheet1"}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got, want := createOut.Resolved.S3Key, "tenants/t1/teams/team1/cases/c1/workspace/report.xlsx"; got != want {
		t.Fatalf("s3Key = %q, want %q", got, want)
	}

	_, err = svc.WriteCells(context.Background(), api.WriteCellsRequest{
		WorkbookBaseRequest: base,
		SheetName:           "Sheet1",
		StartCell:           "A1",
		Values: [][]any{
			{"name", "score"},
			{"alice", 95},
		},
	})
	if err != nil {
		t.Fatalf("WriteCells: %v", err)
	}

	body := store.objects["private-bucket:tenants/t1/teams/team1/cases/c1/workspace/report.xlsx"]
	f, openErr := excelize.OpenReader(bytes.NewReader(body))
	if openErr != nil {
		t.Fatalf("OpenReader: %v", openErr)
	}
	v, cellErr := f.GetCellValue("Sheet1", "B2")
	if cellErr != nil {
		t.Fatalf("GetCellValue: %v", cellErr)
	}
	if got, want := v, "95"; got != want {
		t.Fatalf("B2 = %q, want %q", got, want)
	}
}

func TestExecuteFlow_WriteThenReadDocument(t *testing.T) {
	store := &memoryStore{}
	svc := app.DocumentService{
		Resolver: resolverFunc(vfs.ResolveFile),
		Store:    store,
		Engine:   document.NewEngine(),
	}

	base := api.DocumentBaseRequest{
		VFS: api.VFSContext{
			Mounts: map[string]api.VFSMount{
				"/workdir/": {
					Permission: "read_write",
					Bucket:     "private",
					Path:       "tenants/{tenant_id}/teams/{team_id}/cases/{case_id}/workspace/",
					TTLMS:      30000,
				},
			},
			S3Sets: map[string]api.S3SetConfig{
				"private": {Bucket: "private-bucket"},
			},
			TemplateVars: map[string]string{
				"tenant_id": "t1",
				"team_id":   "team1",
				"case_id":   "c1",
			},
		},
		FilePath: "/workdir/brief.docx",
	}

	writeOut, err := svc.Write(context.Background(), api.WriteDocumentRequest{
		DocumentBaseRequest: base,
		Blocks: []api.DocumentBlock{
			{Type: "title", Text: "Weekly Report"},
			{Type: "heading", Level: 1, Text: "Progress"},
			{Type: "paragraph", Text: "Done."},
			{Type: "table", Rows: [][]any{{"name", "score"}, {"alice", 95}}},
		},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got, want := writeOut.Resolved.S3Key, "tenants/t1/teams/team1/cases/c1/workspace/brief.docx"; got != want {
		t.Fatalf("s3Key = %q, want %q", got, want)
	}

	metaOut, err := svc.Read(context.Background(), api.ReadDocumentRequest{
		DocumentBaseRequest: base,
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if metaOut.Meta.ParagraphCount < 3 {
		t.Fatalf("paragraphCount = %d, want >= 3", metaOut.Meta.ParagraphCount)
	}
	if metaOut.Meta.TableCount != 1 {
		t.Fatalf("tableCount = %d, want 1", metaOut.Meta.TableCount)
	}
}

var _ storage.ObjectStore = (*memoryStore)(nil)
