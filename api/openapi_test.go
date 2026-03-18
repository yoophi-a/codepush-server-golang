package api_test

import (
	"path/filepath"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
)

func TestOpenAPIIsValid(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromFile(filepath.Join("openapi.yaml"))
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}
	if err := doc.Validate(loader.Context); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
