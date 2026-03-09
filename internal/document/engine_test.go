package document

import (
	"testing"

	"github.com/github-flaboy/officeman/internal/api"
)

func TestEngine_WriteDocument(t *testing.T) {
	engine := NewEngine()

	body, meta, err := engine.Write(api.WriteDocumentRequest{
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
	if len(body) == 0 {
		t.Fatal("expected docx bytes")
	}
	if meta.ParagraphCount < 3 {
		t.Fatalf("paragraphCount = %d, want >= 3", meta.ParagraphCount)
	}
	if meta.TableCount != 1 {
		t.Fatalf("tableCount = %d, want 1", meta.TableCount)
	}
}

func TestEngine_Read(t *testing.T) {
	engine := NewEngine()

	body, _, err := engine.Write(api.WriteDocumentRequest{
		Blocks: []api.DocumentBlock{
			{Type: "title", Text: "Weekly Report"},
			{Type: "paragraph", Text: "Done."},
		},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	meta, err := engine.Read(body)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if meta.ParagraphCount < 2 {
		t.Fatalf("paragraphCount = %d, want >= 2", meta.ParagraphCount)
	}
}
