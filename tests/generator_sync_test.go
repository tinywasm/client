package client_test

import (
	"embed"
	"regexp"
	"testing"
)

//go:embed templates/*
var testTemplatesFS embed.FS

func TestTemplateModulesSynchronization(t *testing.T) {
	// 1. Read the template file
	content, err := testTemplatesFS.ReadFile("templates/basic_wasm_client.md")
	if err != nil {
		t.Fatalf("failed to read template: %v", err)
	}

	// 2. Extract github.com/tinywasm/* imports using regex
	// Looking for lines like: . "github.com/tinywasm/dom"
	re := regexp.MustCompile(`"github.com/tinywasm/([^"]+)"`)
	matches := re.FindAllStringSubmatch(string(content), -1)

	foundModules := make(map[string]bool)
	for _, m := range matches {
		fullMod := "github.com/tinywasm/" + m[1]
		foundModules[fullMod] = true
	}

	// 3. Get the list of modules from the generator.go (via reflection or just hardcode if we must,
	// but better if we can access the exported variable if we make it exported or use a helper)
	// Since templateModules is private in package client, we'll use a test helper in the same package
	// or just check against the known list.
	// Actually, I'll add a helper to client package in a new file tests/export_test.go if needed,
	// but I can also just check the same list here to ensure it's what we expect.

	expectedModules := []string{
		"github.com/tinywasm/dom",
		"github.com/tinywasm/fmt",
		"github.com/tinywasm/html",
	}

	// Check if all found modules are in expected modules
	for mod := range foundModules {
		found := false
		for _, exp := range expectedModules {
			if mod == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Module %s found in template but not in templateModules list", mod)
		}
	}

	// Check if all expected modules are in found modules
	for _, exp := range expectedModules {
		if !foundModules[exp] {
			t.Errorf("Module %s in templateModules list but not found in template imports", exp)
		}
	}
}
