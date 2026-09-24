package crud

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateCreatesFormattedCRUDForUUIDModel(t *testing.T) {
	dir := t.TempDir()
	modelPath := filepath.Join(dir, "book.go")
	model := `package books

import (
    "time"
    "github.com/google/uuid"
)

type Book struct {
    ID uuid.UUID ` + "`json:\"id\"`" + `
    Title string ` + "`json:\"title\" validate:\"required\"`" + `
    CreatedAt time.Time ` + "`json:\"created_at\"`" + `
}
`
	if err := os.WriteFile(modelPath, []byte(model), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Generate(Options{ModelPath: modelPath}); err != nil {
		t.Fatal(err)
	}

	controllerPath := filepath.Join(dir, "book_controller_gen.go")
	controller, err := os.ReadFile(controllerPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(controller), "c.Status(http.StatusNoContent)") {
		t.Fatal("generated delete handler does not return an empty 204 response")
	}

	files, err := filepath.Glob(filepath.Join(dir, "*_gen.go"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 5 {
		t.Fatalf("generated file count = %d, want 5", len(files))
	}
	for _, path := range files {
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(contents), "\n\n\n") {
			t.Fatalf("generated file %s contains unformatted blank lines", path)
		}
	}
}
