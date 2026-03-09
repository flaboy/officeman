package vfs

import (
	"testing"

	"github.com/github-flaboy/officeman/internal/api"
)

func TestResolvePath_UsesLongestMountPrefix(t *testing.T) {
	resolved, err := ResolveFile(testVFSContext(), "/workdir/reports/daily.xlsx")
	if err != nil {
		t.Fatalf("ResolveFile: %v", err)
	}

	if got, want := resolved.MountPath, "/workdir/reports/"; got != want {
		t.Fatalf("mountPath = %q, want %q", got, want)
	}
	if got, want := resolved.S3Key, "tenants/t1/teams/team1/cases/case1/workspace/reports/daily.xlsx"; got != want {
		t.Fatalf("s3Key = %q, want %q", got, want)
	}
}

func TestResolvePath_RejectsTraversal(t *testing.T) {
	_, err := ResolveFile(testVFSContext(), "/workdir/../secrets.xlsx")
	if err == nil {
		t.Fatal("expected resolve error")
	}
	if got, want := err.Code, "VFS_PATH_TRAVERSAL"; got != want {
		t.Fatalf("code = %q, want %q", got, want)
	}
}

func TestResolvePath_RendersTemplateVarsIntoS3Key(t *testing.T) {
	resolved, err := ResolveFile(testVFSContext(), "/workdir/report.xlsx")
	if err != nil {
		t.Fatalf("ResolveFile: %v", err)
	}

	if got, want := resolved.BucketAlias, "private"; got != want {
		t.Fatalf("bucketAlias = %q, want %q", got, want)
	}
	if got, want := resolved.BucketName, "private-bucket"; got != want {
		t.Fatalf("bucketName = %q, want %q", got, want)
	}
	if got, want := resolved.S3Prefix, "tenants/t1/teams/team1/cases/case1/workspace/"; got != want {
		t.Fatalf("s3Prefix = %q, want %q", got, want)
	}
	if got, want := resolved.RelativePath, "report.xlsx"; got != want {
		t.Fatalf("relativePath = %q, want %q", got, want)
	}
	if got, want := resolved.S3Key, "tenants/t1/teams/team1/cases/case1/workspace/report.xlsx"; got != want {
		t.Fatalf("s3Key = %q, want %q", got, want)
	}
}

func TestResolvePath_RejectsUnknownBucketAlias(t *testing.T) {
	ctx := testVFSContext()
	ctx.Mounts["/missing/"] = api.VFSMount{
		Permission: "read_write",
		Bucket:     "unknown",
		Path:       "tenants/{tenant_id}/missing/",
		TTLMS:      30000,
	}

	_, err := ResolveFile(ctx, "/missing/report.xlsx")
	if err == nil {
		t.Fatal("expected resolve error")
	}
	if got, want := err.Code, "VFS_S3_SET_NOT_FOUND"; got != want {
		t.Fatalf("code = %q, want %q", got, want)
	}
}

func testVFSContext() api.VFSContext {
	return api.VFSContext{
		Mounts: map[string]api.VFSMount{
			"/workdir/": {
				Permission: "read_write",
				Bucket:     "private",
				Path:       "tenants/{tenant_id}/teams/{team_id}/cases/{case_id}/workspace/",
				TTLMS:      30000,
			},
			"/workdir/reports/": {
				Permission: "read_write",
				Bucket:     "private",
				Path:       "tenants/{tenant_id}/teams/{team_id}/cases/{case_id}/workspace/reports/",
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
	}
}
