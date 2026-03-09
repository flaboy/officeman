package document

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/github-flaboy/officeman/internal/api"
	"github.com/gomutex/godocx"
	docxlib "github.com/gomutex/godocx/docx"
)

type Meta struct {
	ParagraphCount int `json:"paragraphCount"`
	TableCount     int `json:"tableCount"`
}

type Engine interface {
	Write(req api.WriteDocumentRequest) ([]byte, Meta, error)
	Read(src []byte) (Meta, error)
}

type engine struct{}

func NewEngine() Engine {
	return &engine{}
}

func (e *engine) Write(req api.WriteDocumentRequest) ([]byte, Meta, error) {
	doc, err := godocx.NewDocument()
	if err != nil {
		return nil, Meta{}, err
	}

	for _, block := range req.Blocks {
		switch block.Type {
		case "title":
			p := doc.AddParagraph(block.Text)
			p.Style("Title")
		case "heading":
			if _, err := doc.AddHeading(block.Text, uint(block.Level)); err != nil {
				return nil, Meta{}, err
			}
		case "paragraph":
			doc.AddParagraph(block.Text)
		case "table":
			tbl := doc.AddTable()
			for _, row := range block.Rows {
				r := tbl.AddRow()
				for _, cell := range row {
					r.AddCell().AddParagraph(stringify(cell))
				}
			}
		}
	}

	buf := bytes.NewBuffer(nil)
	if _, err := doc.WriteTo(buf); err != nil {
		return nil, Meta{}, err
	}
	meta, err := e.Read(buf.Bytes())
	if err != nil {
		return nil, Meta{}, err
	}
	return buf.Bytes(), meta, nil
}

func (e *engine) Read(src []byte) (Meta, error) {
	tmp, err := os.CreateTemp("", "officeman-*.docx")
	if err != nil {
		return Meta{}, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(src); err != nil {
		tmp.Close()
		return Meta{}, err
	}
	if err := tmp.Close(); err != nil {
		return Meta{}, err
	}

	doc, err := godocx.OpenDocument(tmp.Name())
	if err != nil {
		return Meta{}, err
	}
	return metaFromDoc(doc), nil
}

func metaFromDoc(doc *docxlib.RootDoc) Meta {
	meta := Meta{}
	for _, child := range doc.Document.Body.Children {
		if child.Para != nil {
			meta.ParagraphCount++
		}
		if child.Table != nil {
			meta.TableCount++
		}
	}
	return meta
}

func stringify(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case []byte:
		return string(x)
	default:
		return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(fmt.Sprint(v), "\r\n", "\n"), "\r", "\n"))
	}
}
