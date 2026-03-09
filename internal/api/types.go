package api

type WorkbookBaseRequest struct {
	RequestID string     `json:"requestId,omitempty"`
	VFS       VFSContext `json:"vfs"`
	FilePath  string     `json:"filePath"`
}

type VFSContext struct {
	Mounts       map[string]VFSMount    `json:"mounts"`
	S3Sets       map[string]S3SetConfig `json:"s3_sets"`
	TemplateVars map[string]string      `json:"template_vars,omitempty"`
}

type VFSMount struct {
	Permission string `json:"permission"`
	Bucket     string `json:"bucket"`
	Path       string `json:"path"`
	TTLMS      int64  `json:"ttl_ms"`
}

type S3SetConfig struct {
	Bucket          string `json:"bucket"`
	Endpoint        string `json:"endpoint,omitempty"`
	Region          string `json:"region,omitempty"`
	AccessKeyID     string `json:"access_key_id,omitempty"`
	SecretAccessKey string `json:"secret_access_key,omitempty"`
	ForcePathStyle  bool   `json:"force_path_style,omitempty"`
}

type CreateWorkbookRequest struct {
	WorkbookBaseRequest
	Sheets []CreateWorkbookSheet `json:"sheets,omitempty"`
}

type CreateWorkbookSheet struct {
	Name string  `json:"name"`
	Rows [][]any `json:"rows,omitempty"`
}

type WorkbookMetaRequest struct {
	WorkbookBaseRequest
}

type WriteCellsRequest struct {
	WorkbookBaseRequest
	SheetName string  `json:"sheetName"`
	StartCell string  `json:"startCell"`
	Values    [][]any `json:"values"`
}

type AppendRowsRequest struct {
	WorkbookBaseRequest
	SheetName string  `json:"sheetName"`
	Rows      [][]any `json:"rows"`
}

type AddSheetRequest struct {
	WorkbookBaseRequest
	SheetName string `json:"sheetName"`
}

type RenameSheetRequest struct {
	WorkbookBaseRequest
	FromSheetName string `json:"fromSheetName"`
	ToSheetName   string `json:"toSheetName"`
}

type DeleteSheetRequest struct {
	WorkbookBaseRequest
	SheetName string `json:"sheetName"`
}

type DocumentBaseRequest struct {
	RequestID string     `json:"requestId,omitempty"`
	VFS       VFSContext `json:"vfs"`
	FilePath  string     `json:"filePath"`
}

type WriteDocumentRequest struct {
	DocumentBaseRequest
	Blocks []DocumentBlock `json:"blocks,omitempty"`
}

type DocumentBlock struct {
	Type  string  `json:"type"`
	Text  string  `json:"text,omitempty"`
	Level int     `json:"level,omitempty"`
	Rows  [][]any `json:"rows,omitempty"`
}

type ReadDocumentRequest struct {
	DocumentBaseRequest
}

type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

func (r WorkbookBaseRequest) Validate() *ValidationError {
	if r.FilePath == "" {
		return invalid("filePath is required")
	}
	if r.FilePath[0] != '/' {
		return invalid("filePath must be an absolute vfs path")
	}
	if len(r.FilePath) < len(".xlsx") || r.FilePath[len(r.FilePath)-len(".xlsx"):] != ".xlsx" {
		return invalid("filePath must end with .xlsx")
	}
	if len(r.VFS.Mounts) == 0 {
		return invalid("vfs.mounts is required")
	}
	if len(r.VFS.S3Sets) == 0 {
		return invalid("vfs.s3_sets is required")
	}
	return nil
}

func (r CreateWorkbookRequest) Validate() *ValidationError {
	if err := r.WorkbookBaseRequest.Validate(); err != nil {
		return err
	}
	for _, sheet := range r.Sheets {
		if sheet.Name == "" {
			return invalid("sheet name is required")
		}
	}
	return nil
}

func (r WorkbookMetaRequest) Validate() *ValidationError {
	return r.WorkbookBaseRequest.Validate()
}

func (r WriteCellsRequest) Validate() *ValidationError {
	if err := r.WorkbookBaseRequest.Validate(); err != nil {
		return err
	}
	if r.SheetName == "" {
		return invalid("sheetName is required")
	}
	if r.StartCell == "" {
		return invalid("startCell is required")
	}
	if len(r.Values) == 0 {
		return invalid("values is required")
	}
	return nil
}

func (r AppendRowsRequest) Validate() *ValidationError {
	if err := r.WorkbookBaseRequest.Validate(); err != nil {
		return err
	}
	if r.SheetName == "" {
		return invalid("sheetName is required")
	}
	if len(r.Rows) == 0 {
		return invalid("rows is required")
	}
	return nil
}

func (r AddSheetRequest) Validate() *ValidationError {
	if err := r.WorkbookBaseRequest.Validate(); err != nil {
		return err
	}
	if r.SheetName == "" {
		return invalid("sheetName is required")
	}
	return nil
}

func (r RenameSheetRequest) Validate() *ValidationError {
	if err := r.WorkbookBaseRequest.Validate(); err != nil {
		return err
	}
	if r.FromSheetName == "" || r.ToSheetName == "" {
		return invalid("fromSheetName and toSheetName are required")
	}
	return nil
}

func (r DeleteSheetRequest) Validate() *ValidationError {
	if err := r.WorkbookBaseRequest.Validate(); err != nil {
		return err
	}
	if r.SheetName == "" {
		return invalid("sheetName is required")
	}
	return nil
}

func (r DocumentBaseRequest) Validate() *ValidationError {
	if r.FilePath == "" {
		return invalid("filePath is required")
	}
	if r.FilePath[0] != '/' {
		return invalid("filePath must be an absolute document path")
	}
	if len(r.FilePath) < len(".docx") || r.FilePath[len(r.FilePath)-len(".docx"):] != ".docx" {
		return invalid("filePath must end with .docx")
	}
	if len(r.VFS.Mounts) == 0 {
		return invalid("mounts are required")
	}
	if len(r.VFS.S3Sets) == 0 {
		return invalid("storage sets are required")
	}
	return nil
}

func (r WriteDocumentRequest) Validate() *ValidationError {
	if err := r.DocumentBaseRequest.Validate(); err != nil {
		return err
	}
	if len(r.Blocks) == 0 {
		return invalid("blocks are required")
	}
	for _, block := range r.Blocks {
		switch block.Type {
		case "title", "paragraph":
			if block.Text == "" {
				return invalid("text is required")
			}
		case "heading":
			if block.Text == "" {
				return invalid("text is required")
			}
			if block.Level != 1 && block.Level != 2 {
				return invalid("heading level must be 1 or 2")
			}
		case "table":
			if len(block.Rows) == 0 {
				return invalid("rows are required")
			}
		default:
			return invalid("block type is invalid")
		}
	}
	return nil
}

func (r ReadDocumentRequest) Validate() *ValidationError {
	return r.DocumentBaseRequest.Validate()
}

func invalid(message string) *ValidationError {
	return &ValidationError{Code: "INVALID_REQUEST", Message: message}
}
