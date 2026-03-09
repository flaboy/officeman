package excel

import (
	"bytes"
	"fmt"

	"github.com/github-flaboy/officeman/internal/api"
	"github.com/xuri/excelize/v2"
)

type WorkbookMeta struct {
	Sheets []SheetMeta `json:"sheets"`
}

type SheetMeta struct {
	Name     string `json:"name"`
	LastCell string `json:"lastCell,omitempty"`
}

type Engine interface {
	CreateWorkbook(req api.CreateWorkbookRequest) ([]byte, WorkbookMeta, error)
	ReadWorkbookMeta(src []byte) (WorkbookMeta, error)
	WriteCells(src []byte, req api.WriteCellsRequest) ([]byte, WorkbookMeta, error)
	AppendRows(src []byte, req api.AppendRowsRequest) ([]byte, WorkbookMeta, error)
	AddSheet(src []byte, req api.AddSheetRequest) ([]byte, WorkbookMeta, error)
	RenameSheet(src []byte, req api.RenameSheetRequest) ([]byte, WorkbookMeta, error)
	DeleteSheet(src []byte, req api.DeleteSheetRequest) ([]byte, WorkbookMeta, error)
}

type engine struct{}

func NewEngine() Engine {
	return &engine{}
}

func (e *engine) CreateWorkbook(req api.CreateWorkbookRequest) ([]byte, WorkbookMeta, error) {
	f := excelize.NewFile()
	sheets := req.Sheets
	if len(sheets) == 0 {
		sheets = []api.CreateWorkbookSheet{{Name: "Sheet1"}}
	}

	firstName := sheets[0].Name
	if firstName == "" {
		firstName = "Sheet1"
	}
	if f.GetSheetName(0) != firstName {
		f.SetSheetName(f.GetSheetName(0), firstName)
	}
	for _, row := range sheets[0].Rows {
		if err := appendRow(f, firstName, row); err != nil {
			return nil, WorkbookMeta{}, err
		}
	}
	for i := 1; i < len(sheets); i++ {
		name := sheets[i].Name
		if name == "" {
			return nil, WorkbookMeta{}, fmt.Errorf("sheet name is required")
		}
		f.NewSheet(name)
		for _, row := range sheets[i].Rows {
			if err := appendRow(f, name, row); err != nil {
				return nil, WorkbookMeta{}, err
			}
		}
	}
	return writeWorkbook(f)
}

func (e *engine) ReadWorkbookMeta(src []byte) (WorkbookMeta, error) {
	f, err := openWorkbook(src)
	if err != nil {
		return WorkbookMeta{}, err
	}
	return workbookMeta(f), nil
}

func (e *engine) WriteCells(src []byte, req api.WriteCellsRequest) ([]byte, WorkbookMeta, error) {
	f, err := openWorkbook(src)
	if err != nil {
		return nil, WorkbookMeta{}, err
	}
	if len(req.Values) == 0 {
		return nil, WorkbookMeta{}, fmt.Errorf("values is required")
	}
	currentCell := req.StartCell
	for _, row := range req.Values {
		rowCopy := append([]any(nil), row...)
		if err := f.SetSheetRow(req.SheetName, currentCell, &rowCopy); err != nil {
			return nil, WorkbookMeta{}, err
		}
		col, line, err := excelize.CellNameToCoordinates(currentCell)
		if err != nil {
			return nil, WorkbookMeta{}, err
		}
		next, err := excelize.CoordinatesToCellName(col, line+1)
		if err != nil {
			return nil, WorkbookMeta{}, err
		}
		currentCell = next
	}
	return writeWorkbook(f)
}

func (e *engine) AppendRows(src []byte, req api.AppendRowsRequest) ([]byte, WorkbookMeta, error) {
	f, err := openWorkbook(src)
	if err != nil {
		return nil, WorkbookMeta{}, err
	}
	rows, err := f.GetRows(req.SheetName)
	if err != nil {
		return nil, WorkbookMeta{}, err
	}
	startRow := len(rows) + 1
	for idx, row := range req.Rows {
		rowCopy := append([]any(nil), row...)
		cell, err := excelize.CoordinatesToCellName(1, startRow+idx)
		if err != nil {
			return nil, WorkbookMeta{}, err
		}
		if err := f.SetSheetRow(req.SheetName, cell, &rowCopy); err != nil {
			return nil, WorkbookMeta{}, err
		}
	}
	return writeWorkbook(f)
}

func (e *engine) AddSheet(src []byte, req api.AddSheetRequest) ([]byte, WorkbookMeta, error) {
	f, err := openWorkbook(src)
	if err != nil {
		return nil, WorkbookMeta{}, err
	}
	f.NewSheet(req.SheetName)
	return writeWorkbook(f)
}

func (e *engine) RenameSheet(src []byte, req api.RenameSheetRequest) ([]byte, WorkbookMeta, error) {
	f, err := openWorkbook(src)
	if err != nil {
		return nil, WorkbookMeta{}, err
	}
	if err := f.SetSheetName(req.FromSheetName, req.ToSheetName); err != nil {
		return nil, WorkbookMeta{}, err
	}
	return writeWorkbook(f)
}

func (e *engine) DeleteSheet(src []byte, req api.DeleteSheetRequest) ([]byte, WorkbookMeta, error) {
	f, err := openWorkbook(src)
	if err != nil {
		return nil, WorkbookMeta{}, err
	}
	if err := f.DeleteSheet(req.SheetName); err != nil {
		return nil, WorkbookMeta{}, err
	}
	return writeWorkbook(f)
}

func openWorkbook(src []byte) (*excelize.File, error) {
	return excelize.OpenReader(bytes.NewReader(src))
}

func appendRow(f *excelize.File, sheet string, row []any) error {
	rows, err := f.GetRows(sheet)
	if err != nil {
		return err
	}
	cell, err := excelize.CoordinatesToCellName(1, len(rows)+1)
	if err != nil {
		return err
	}
	rowCopy := append([]any(nil), row...)
	return f.SetSheetRow(sheet, cell, &rowCopy)
}

func writeWorkbook(f *excelize.File) ([]byte, WorkbookMeta, error) {
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, WorkbookMeta{}, err
	}
	return buf.Bytes(), workbookMeta(f), nil
}

func workbookMeta(f *excelize.File) WorkbookMeta {
	sheets := make([]SheetMeta, 0, len(f.GetSheetList()))
	for _, name := range f.GetSheetList() {
		lastCell := ""
		rows, err := f.GetRows(name)
		if err == nil && len(rows) > 0 {
			lastRow := len(rows)
			lastCol := 1
			for _, row := range rows {
				if len(row) > lastCol {
					lastCol = len(row)
				}
			}
			if cell, cellErr := excelize.CoordinatesToCellName(lastCol, lastRow); cellErr == nil {
				lastCell = cell
			}
		}
		sheets = append(sheets, SheetMeta{Name: name, LastCell: lastCell})
	}
	return WorkbookMeta{Sheets: sheets}
}
