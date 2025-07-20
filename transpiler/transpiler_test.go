package transpiler

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aisk/ego/internal/diff"
)

func TestTranspiler(t *testing.T) {
	testdata := "testdata"
	
	entries, err := os.ReadDir(testdata)
	if err != nil {
		t.Fatalf("Failed to read testdata directory: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".ego") {
			egoFile := filepath.Join(testdata, entry.Name())
			expectedFile := filepath.Join(testdata, strings.TrimSuffix(entry.Name(), ".ego")+"_expected.go")
			
			t.Run(entry.Name(), func(t *testing.T) {
				// Read .ego file
				egoContent, err := os.ReadFile(egoFile)
				if err != nil {
					t.Fatalf("Failed to read .ego file: %v", err)
				}
				
				// Read expected output
				expectedContent, err := os.ReadFile(expectedFile)
				if err != nil {
					t.Fatalf("Failed to read expected.go file: %v", err)
				}
				
				// Transpile the .ego content
				input := bytes.NewReader(egoContent)
				var output bytes.Buffer
				
				err = Transpile(input, &output)
				if err != nil {
					t.Fatalf("Transpile failed: %v", err)
				}
				
				// Compare with expected
				if output.String() != string(expectedContent) {
					diffOutput := diff.Diff(expectedFile, expectedContent, "transpiled", []byte(output.String()))
					t.Errorf("Transpiled result does not match expected:\n%s", diffOutput)
				}
			})
		}
	}
}