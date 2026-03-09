package app

import (
	"context"
	"testing"

	"github.com/github-flaboy/officeman/internal/api"
	"github.com/github-flaboy/officeman/internal/excel"
	"github.com/github-flaboy/officeman/internal/vfs"
)

type fakeResolver struct {
	resolved vfs.ResolvedFile
	err      *vfs.ResolveError
}

func (f fakeResolver) ResolveFile(_ api.VFSContext, _ string) (vfs.ResolvedFile, *vfs.ResolveError) {
	if f.err != nil {
		return vfs.ResolvedFile{}, f.err
	}
	return f.resolved, nil
}

type fakeStore struct {
	headExists bool
	getBody    []byte
	putKey     string
	putBucket  string
	putBody    []byte
	putCalls   int
}

func (f *fakeStore) GetObjectBytes(_ context.Context, cfg api.S3SetConfig, key string) ([]byte, error) {
	f.putBucket = cfg.Bucket
	f.putKey = key
	return f.getBody, nil
}

func (f *fakeStore) PutObjectBytes(_ context.Context, cfg api.S3SetConfig, key string, body []byte, _ string) error {
	f.putBucket = cfg.Bucket
	f.putKey = key
	f.putBody = body
	f.putCalls++
	return nil
}

func (f *fakeStore) HeadObject(_ context.Context, _ api.S3SetConfig, _ string) (bool, error) {
	return f.headExists, nil
}

type fakeEngine struct {
	createBytes []byte
	meta        excel.WorkbookMeta
	readMeta    excel.WorkbookMeta
}

func (f fakeEngine) CreateWorkbook(_ api.CreateWorkbookRequest) ([]byte, excel.WorkbookMeta, error) {
	return f.createBytes, f.meta, nil
}

func (f fakeEngine) ReadWorkbookMeta(_ []byte) (excel.WorkbookMeta, error) {
	return f.readMeta, nil
}

func (f fakeEngine) WriteCells(_ []byte, _ api.WriteCellsRequest) ([]byte, excel.WorkbookMeta, error) {
	return []byte("write-cells"), f.meta, nil
}

func (f fakeEngine) AppendRows(_ []byte, _ api.AppendRowsRequest) ([]byte, excel.WorkbookMeta, error) {
	return []byte("append-rows"), f.meta, nil
}

func (f fakeEngine) AddSheet(_ []byte, _ api.AddSheetRequest) ([]byte, excel.WorkbookMeta, error) {
	return []byte("add-sheet"), f.meta, nil
}

func (f fakeEngine) RenameSheet(_ []byte, _ api.RenameSheetRequest) ([]byte, excel.WorkbookMeta, error) {
	return []byte("rename-sheet"), f.meta, nil
}

func (f fakeEngine) DeleteSheet(_ []byte, _ api.DeleteSheetRequest) ([]byte, excel.WorkbookMeta, error) {
	return []byte("delete-sheet"), f.meta, nil
}

func TestWorkbookService_CreateWorkbookToResolvedS3Key(t *testing.T) {
	store := &fakeStore{}
	svc := WorkbookService{
		Resolver: fakeResolver{
			resolved: vfs.ResolvedFile{
				Permission:  "read_write",
				BucketAlias: "private",
				BucketName:  "private-bucket",
				S3Key:       "tenants/t1/teams/team1/cases/case1/workspace/report.xlsx",
			},
		},
		Store: store,
		Engine: fakeEngine{
			createBytes: []byte("xlsx"),
			meta:        excel.WorkbookMeta{Sheets: []excel.SheetMeta{{Name: "Sheet1"}}},
		},
	}

	out, err := svc.Create(context.Background(), api.CreateWorkbookRequest{
		WorkbookBaseRequest: validBaseRequest(),
		Sheets:              []api.CreateWorkbookSheet{{Name: "Sheet1"}},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if got, want := out.Resolved.S3Key, "tenants/t1/teams/team1/cases/case1/workspace/report.xlsx"; got != want {
		t.Fatalf("s3Key = %q, want %q", got, want)
	}
	if got, want := store.putBucket, "private-bucket"; got != want {
		t.Fatalf("bucket = %q, want %q", got, want)
	}
	if got, want := string(store.putBody), "xlsx"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestWorkbookService_RejectsWriteOnReadOnlyMount(t *testing.T) {
	svc := WorkbookService{
		Resolver: fakeResolver{
			resolved: vfs.ResolvedFile{
				Permission:  "read_only",
				BucketAlias: "private",
				BucketName:  "private-bucket",
				S3Key:       "tenants/t1/teams/team1/cases/case1/workspace/report.xlsx",
			},
		},
		Store:  &fakeStore{},
		Engine: fakeEngine{},
	}

	_, err := svc.WriteCells(context.Background(), api.WriteCellsRequest{
		WorkbookBaseRequest: validBaseRequest(),
		SheetName:           "Sheet1",
		StartCell:           "A1",
		Values:              [][]any{{"name"}},
	})
	if err == nil {
		t.Fatal("expected service error")
	}
	if got, want := err.Code, "VFS_PERMISSION_DENIED"; got != want {
		t.Fatalf("code = %q, want %q", got, want)
	}
}

func TestWorkbookService_ReadWorkbookMetaDoesNotWriteBack(t *testing.T) {
	store := &fakeStore{getBody: []byte("xlsx")}
	svc := WorkbookService{
		Resolver: fakeResolver{
			resolved: vfs.ResolvedFile{
				Permission:  "read_write",
				BucketAlias: "private",
				BucketName:  "private-bucket",
				S3Key:       "tenants/t1/teams/team1/cases/case1/workspace/report.xlsx",
			},
		},
		Store: store,
		Engine: fakeEngine{
			readMeta: excel.WorkbookMeta{Sheets: []excel.SheetMeta{{Name: "Sheet1"}}},
		},
	}

	out, err := svc.Meta(context.Background(), api.WorkbookMetaRequest{
		WorkbookBaseRequest: validBaseRequest(),
	})
	if err != nil {
		t.Fatalf("Meta: %v", err)
	}
	if store.putCalls != 0 {
		t.Fatalf("putCalls = %d, want 0", store.putCalls)
	}
	if len(out.Meta.Sheets) != 1 || out.Meta.Sheets[0].Name != "Sheet1" {
		t.Fatalf("meta = %+v", out.Meta)
	}
}

func validBaseRequest() api.WorkbookBaseRequest {
	return api.WorkbookBaseRequest{
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
				"case_id":   "case1",
			},
		},
		FilePath: "/workdir/report.xlsx",
	}
}
