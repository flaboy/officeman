package app

import (
	"context"
	"testing"

	"github.com/github-flaboy/officeman/internal/api"
	"github.com/github-flaboy/officeman/internal/document"
	"github.com/github-flaboy/officeman/internal/vfs"
)

type fakeDocumentEngine struct {
	body []byte
	meta document.Meta
}

func (f fakeDocumentEngine) Write(_ api.WriteDocumentRequest) ([]byte, document.Meta, error) {
	return f.body, f.meta, nil
}

func (f fakeDocumentEngine) Read(_ []byte) (document.Meta, error) {
	return f.meta, nil
}

func TestDocumentService_WriteDocumentToResolvedS3Key(t *testing.T) {
	store := &fakeStore{}
	svc := DocumentService{
		Resolver: fakeResolver{
			resolved: vfs.ResolvedFile{
				Permission:  "read_write",
				BucketAlias: "private",
				BucketName:  "private-bucket",
				S3Key:       "tenants/t1/teams/team1/cases/case1/workspace/brief.docx",
			},
		},
		Store: store,
		Engine: fakeDocumentEngine{
			body: []byte("docx"),
			meta: document.Meta{ParagraphCount: 2},
		},
	}

	out, err := svc.Write(context.Background(), api.WriteDocumentRequest{
		DocumentBaseRequest: validDocumentBaseRequest(),
		Blocks:              []api.DocumentBlock{{Type: "title", Text: "Weekly Report"}},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if got, want := out.Resolved.S3Key, "tenants/t1/teams/team1/cases/case1/workspace/brief.docx"; got != want {
		t.Fatalf("s3Key = %q, want %q", got, want)
	}
	if got, want := string(store.putBody), "docx"; got != want {
		t.Fatalf("body = %q, want %q", got, want)
	}
}

func TestDocumentService_RejectsWriteOnReadOnlyMount(t *testing.T) {
	svc := DocumentService{
		Resolver: fakeResolver{
			resolved: vfs.ResolvedFile{
				Permission:  "read_only",
				BucketAlias: "private",
				BucketName:  "private-bucket",
				S3Key:       "tenants/t1/teams/team1/cases/case1/workspace/brief.docx",
			},
		},
		Store:  &fakeStore{},
		Engine: fakeDocumentEngine{},
	}

	_, err := svc.Write(context.Background(), api.WriteDocumentRequest{
		DocumentBaseRequest: validDocumentBaseRequest(),
		Blocks:              []api.DocumentBlock{{Type: "title", Text: "Weekly Report"}},
	})
	if err == nil {
		t.Fatal("expected service error")
	}
	if got, want := err.Code, "VFS_PERMISSION_DENIED"; got != want {
		t.Fatalf("code = %q, want %q", got, want)
	}
}

func TestDocumentService_ReadDoesNotWriteBack(t *testing.T) {
	store := &fakeStore{getBody: []byte("docx")}
	svc := DocumentService{
		Resolver: fakeResolver{
			resolved: vfs.ResolvedFile{
				Permission:  "read_write",
				BucketAlias: "private",
				BucketName:  "private-bucket",
				S3Key:       "tenants/t1/teams/team1/cases/case1/workspace/brief.docx",
			},
		},
		Store: store,
		Engine: fakeDocumentEngine{
			meta: document.Meta{ParagraphCount: 2},
		},
	}

	out, err := svc.Read(context.Background(), api.ReadDocumentRequest{
		DocumentBaseRequest: validDocumentBaseRequest(),
	})
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if store.putCalls != 0 {
		t.Fatalf("putCalls = %d, want 0", store.putCalls)
	}
	if got, want := out.Meta.ParagraphCount, 2; got != want {
		t.Fatalf("paragraphCount = %d, want %d", got, want)
	}
}

func validDocumentBaseRequest() api.DocumentBaseRequest {
	return api.DocumentBaseRequest{
		VFS:      validBaseRequest().VFS,
		FilePath: "/workdir/brief.docx",
	}
}
