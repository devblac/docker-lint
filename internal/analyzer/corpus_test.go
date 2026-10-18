package analyzer

import (
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/devblac/docker-lint/internal/parser"
)

var updateCorpus = flag.Bool("update-corpus", false, "rewrite the testdata/corpus golden files")

// corpusDir holds real-world Dockerfiles; see testdata/corpus/SOURCES.md.
const corpusDir = "../../testdata/corpus"

// TestCorpus runs the default rule set over every real-world Dockerfile in the
// corpus and compares the findings with the golden file stored next to it.
//
// Any change in what docker-lint reports on real Dockerfiles - a new false
// positive, a lost true positive, a moved line - shows up here as a diff. When
// the change is intended, regenerate the golden files with:
//
//	go test ./internal/analyzer -run TestCorpus -update-corpus
func TestCorpus(t *testing.T) {
	files := corpusFiles(t)
	if len(files) == 0 {
		t.Fatalf("no Dockerfiles found under %s", corpusDir)
	}

	for _, path := range files {
		path := path
		name, err := filepath.Rel(corpusDir, path)
		if err != nil {
			t.Fatalf("relative path for %s: %v", path, err)
		}

		t.Run(filepath.ToSlash(name), func(t *testing.T) {
			got := renderCorpusFindings(t, path)
			goldenPath := path + ".golden"

			if *updateCorpus {
				if err := os.WriteFile(goldenPath, []byte(got), 0o644); err != nil {
					t.Fatalf("write golden file: %v", err)
				}
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("read golden file (generate it with -update-corpus): %v", err)
			}

			if got != normalizeNewlines(string(want)) {
				t.Errorf("findings for %s changed.\n--- golden\n%s--- got\n%s"+
					"If this is intended, rerun with -update-corpus and review the diff.",
					name, normalizeNewlines(string(want)), got)
			}
		})
	}
}

// corpusFiles returns every corpus Dockerfile, sorted for stable subtest order.
func corpusFiles(t *testing.T) []string {
	t.Helper()

	var files []string
	err := filepath.WalkDir(corpusDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(path, ".Dockerfile") {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", corpusDir, err)
	}

	sort.Strings(files)
	return files
}

// renderCorpusFindings parses and analyzes a corpus file and renders its
// findings one per line. A corpus file that fails to parse is a regression:
// these are Dockerfiles that build with Docker today.
func renderCorpusFindings(t *testing.T, path string) string {
	t.Helper()

	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	df, err := parser.ParseString(normalizeNewlines(string(src)))
	if err != nil {
		t.Fatalf("real-world Dockerfile failed to parse: %v", err)
	}

	findings := NewWithDefaults(Config{}).Analyze(df)
	if len(findings) == 0 {
		return "(no findings)\n"
	}

	var b strings.Builder
	for _, f := range findings {
		fmt.Fprintf(&b, "%d:%d %s %s %s\n", f.Line, f.Column, f.Severity, f.RuleID, f.Message)
	}
	return b.String()
}

// normalizeNewlines converts CRLF line endings to LF so results do not depend
// on how the corpus was checked out.
func normalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}
