package excel

import (
	"bytes"
	"testing"

	"github.com/github-flaboy/officeman/internal/api"
	"github.com/xuri/excelize/v2"
)

func TestEngine_CreateWorkbook(t *testing.T) {
	engine := NewEngine()

	out, meta, err := engine.CreateWorkbook(api.CreateWorkbookRequest{
		Sheets: []api.CreateWorkbookSheet{
			{Name: "Sheet1"},
			{Name: "Summary"},
		},
	})
	if err != nil {
		t.Fatalf("CreateWorkbook: %v", err)
	}

	names := sheetNamesFromBytes(t, out)
	if len(names) != 2 || names[0] != "Sheet1" || names[1] != "Summary" {
		t.Fatalf("sheetNames = %v", names)
	}
	if len(meta.Sheets) != 2 {
		t.Fatalf("meta.Sheets = %d, want 2", len(meta.Sheets))
	}
}

func TestEngine_WriteCells(t *testing.T) {
	engine := NewEngine()
	src := workbookBytes(t, func(f *excelize.File) {})

	out, _, err := engine.WriteCells(src, api.WriteCellsRequest{
		SheetName: "Sheet1",
		StartCell: "A1",
		Values: [][]any{
			{"name", "score"},
			{"alice", 95},
		},
	})
	if err != nil {
		t.Fatalf("WriteCells: %v", err)
	}

	f := mustOpenWorkbook(t, out)
	v, err := f.GetCellValue("Sheet1", "B2")
	if err != nil {
		t.Fatalf("GetCellValue: %v", err)
	}
	if got, want := v, "95"; got != want {
		t.Fatalf("B2 = %q, want %q", got, want)
	}
}

func TestEngine_AppendRows(t *testing.T) {
	engine := NewEngine()
	src := workbookBytes(t, func(f *excelize.File) {
		_ = f.SetSheetRow("Sheet1", "A1", &[]any{"name", "score"})
		_ = f.SetSheetRow("Sheet1", "A2", &[]any{"alice", 95})
	})

	out, _, err := engine.AppendRows(src, api.AppendRowsRequest{
		SheetName: "Sheet1",
		Rows: [][]any{
			{"bob", 88},
		},
	})
	if err != nil {
		t.Fatalf("AppendRows: %v", err)
	}

	f := mustOpenWorkbook(t, out)
	v, err := f.GetCellValue("Sheet1", "A3")
	if err != nil {
		t.Fatalf("GetCellValue: %v", err)
	}
	if got, want := v, "bob"; got != want {
		t.Fatalf("A3 = %q, want %q", got, want)
	}
}

func TestEngine_RenameDeleteAndAddSheet(t *testing.T) {
	engine := NewEngine()
	src := workbookBytes(t, func(f *excelize.File) {
		f.NewSheet("Summary")
	})

	withAdded, _, err := engine.AddSheet(src, api.AddSheetRequest{SheetName: "Input"})
	if err != nil {
		t.Fatalf("AddSheet: %v", err)
	}
	withRenamed, _, err := engine.RenameSheet(withAdded, api.RenameSheetRequest{
		FromSheetName: "Input",
		ToSheetName:   "Data",
	})
	if err != nil {
		t.Fatalf("RenameSheet: %v", err)
	}
	withDeleted, _, err := engine.DeleteSheet(withRenamed, api.DeleteSheetRequest{SheetName: "Summary"})
	if err != nil {
		t.Fatalf("DeleteSheet: %v", err)
	}

	names := sheetNamesFromBytes(t, withDeleted)
	if len(names) != 2 || names[0] != "Sheet1" || names[1] != "Data" {
		t.Fatalf("sheetNames = %v", names)
	}
}

func TestEngine_ReadWorkbookMeta(t *testing.T) {
	engine := NewEngine()
	src := workbookBytes(t, func(f *excelize.File) {
		_ = f.SetSheetRow("Sheet1", "A1", &[]any{"name", "score"})
		_ = f.SetSheetRow("Sheet1", "A2", &[]any{"alice", 95})
	})

	meta, err := engine.ReadWorkbookMeta(src)
	if err != nil {
		t.Fatalf("ReadWorkbookMeta: %v", err)
	}
	if len(meta.Sheets) != 1 {
		t.Fatalf("meta.Sheets = %d, want 1", len(meta.Sheets))
	}
	if got, want := meta.Sheets[0].Name, "Sheet1"; got != want {
		t.Fatalf("sheet name = %q, want %q", got, want)
	}
	if got, want := meta.Sheets[0].LastCell, "B2"; got != want {
		t.Fatalf("lastCell = %q, want %q", got, want)
	}
}

func workbookBytes(t *testing.T, mutate func(f *excelize.File)) []byte {
	t.Helper()
	f := excelize.NewFile()
	mutate(f)
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("WriteToBuffer: %v", err)
	}
	return buf.Bytes()
}

func mustOpenWorkbook(t *testing.T, body []byte) *excelize.File {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(body))
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	return f
}

func sheetNamesFromBytes(t *testing.T, body []byte) []string {
	t.Helper()
	return mustOpenWorkbook(t, body).GetSheetList()
}
