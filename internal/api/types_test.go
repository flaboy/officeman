package api

import "testing"

func TestWorkbookBaseRequest_ValidateRejectsNonAbsolutePath(t *testing.T) {
	req := WorkbookBaseRequest{
		VFS: VFSContext{
			Mounts: map[string]VFSMount{
				"/workdir/": {
					Permission: "read_write",
					Bucket:     "private",
					Path:       "tenants/{tenant_id}/teams/{team_id}/cases/{case_id}/workspace/",
					TTLMS:      30000,
				},
			},
			S3Sets: map[string]S3SetConfig{
				"private": {Bucket: "private"},
			},
		},
		FilePath: "report.xlsx",
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if got, want := err.Code, "INVALID_REQUEST"; got != want {
		t.Fatalf("code = %q, want %q", got, want)
	}
}

func TestWriteCellsRequest_ValidateRequiresSheetName(t *testing.T) {
	req := WriteCellsRequest{
		WorkbookBaseRequest: validBaseRequest(),
		StartCell:           "A1",
		Values:              [][]any{{"name", "score"}},
	}

	err := req.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if got, want := err.Code, "INVALID_REQUEST"; got != want {
		t.Fatalf("code = %q, want %q", got, want)
	}
}

func TestCreateWorkbookRequest_ValidateAcceptsSheets(t *testing.T) {
	req := CreateWorkbookRequest{
		WorkbookBaseRequest: validBaseRequest(),
		Sheets: []CreateWorkbookSheet{
			{Name: "Sheet1"},
			{Name: "Summary"},
		},
	}

	if err := req.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
}

func validBaseRequest() WorkbookBaseRequest {
	return WorkbookBaseRequest{
		VFS: VFSContext{
			Mounts: map[string]VFSMount{
				"/workdir/": {
					Permission: "read_write",
					Bucket:     "private",
					Path:       "tenants/{tenant_id}/teams/{team_id}/cases/{case_id}/workspace/",
					TTLMS:      30000,
				},
			},
			S3Sets: map[string]S3SetConfig{
				"private": {Bucket: "private"},
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
