package world_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestCoreDoesNotImportJava verifies the architectural rule that no package
// under core/ may import any package under java/.
//
// This ensures the game core stays edition-agnostic so that a future Bedrock
// adapter can consume the same canonical types without pulling in Java-specific
// code.  The test fails at build time if a developer accidentally adds such an
// import, giving an early, actionable error message.
func TestCoreDoesNotImportJava(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed — cannot locate core/ directory")
	}
	// thisFile = .../core/world/arch_test.go
	// coreDir  = .../core/
	coreDir := filepath.Dir(filepath.Dir(thisFile))

	fset := token.NewFileSet()
	var violations []string

	err := filepath.WalkDir(coreDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}

		f, parseErr := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if parseErr != nil {
			// Skip files that don't parse (e.g. generated or vendor files).
			return nil
		}

		for _, imp := range f.Imports {
			// imp.Path.Value includes surrounding quotes.
			importPath := strings.Trim(imp.Path.Value, `"`)
			if strings.HasPrefix(importPath, "GoCraft/java") {
				rel, _ := filepath.Rel(coreDir, path)
				violations = append(violations,
					"  "+rel+" imports "+importPath)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walking core/ directory %q: %v", coreDir, err)
	}

	if len(violations) > 0 {
		t.Errorf("architectural violation: core/ must not import java/\n%s",
			strings.Join(violations, "\n"))
	}
}
