package files

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestIsDockerfile(t *testing.T) {
	tests := map[string]bool{
		"Dockerfile":              true,
		"dockerfile":              true,
		"Containerfile":           true,
		"Dockerfile.prod":         true,
		"api.Dockerfile":          true,
		"api.dockerfile":          true,
		"Containerfile.dev":       true,
		"Dockerfile.dockerignore": false,
		".dockerignore":           false,
		"Dockerfiles":             false,
		"MyDockerfile":            false,
		"docker-compose.yml":      false,
	}
	for name, want := range tests {
		if got := IsDockerfile(name); got != want {
			t.Errorf("IsDockerfile(%q) = %v, want %v", name, got, want)
		}
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		pattern, path string
		want          bool
	}{
		{"Dockerfile", "Dockerfile", true},
		{"*/Dockerfile", "api/Dockerfile", true},
		{"*/Dockerfile", "services/api/Dockerfile", false},
		{"**/Dockerfile", "Dockerfile", true},
		{"**/Dockerfile", "services/api/Dockerfile", true},
		{"services/**/Dockerfile", "services/Dockerfile", true},
		{"services/**/Dockerfile", "services/a/b/Dockerfile", true},
		{"services/**/Dockerfile", "other/a/Dockerfile", false},
		{"services/**", "services/a/b/Dockerfile", true},
		{"**/*.Dockerfile", "a/b/api.Dockerfile", true},
		{"**/**/x", "x", true},
		{"./api/*", "api/Dockerfile", true},
		{"api/*", "./api/Dockerfile", true},
		{"[", "[", false}, // malformed pattern never matches
	}
	for _, tt := range tests {
		if got := Match(tt.pattern, tt.path); got != tt.want {
			t.Errorf("Match(%q, %q) = %v, want %v", tt.pattern, tt.path, got, tt.want)
		}
	}
}

func TestExcluded(t *testing.T) {
	tests := []struct {
		path     string
		patterns []string
		want     bool
	}{
		{"vendor/x/Dockerfile", []string{"vendor"}, true},       // name anywhere
		{"a/vendor/Dockerfile", []string{"vendor"}, true},       // name anywhere
		{"a/Dockerfile.test", []string{"*.test"}, true},         // basename glob
		{"examples/a/Dockerfile", []string{"examples/"}, true},  // directory with trailing slash
		{"examples/a/Dockerfile", []string{"examples/*"}, true}, // parent directory matched
		{"a/examples/Dockerfile", []string{"examples/*"}, false},
		{"test/fixtures/Dockerfile", []string{"**/fixtures"}, true},
		{"services/api/Dockerfile", []string{"vendor", "examples/"}, false},
		{"./vendor/Dockerfile", []string{"./vendor"}, true},
	}
	for _, tt := range tests {
		if got := Excluded(tt.path, tt.patterns); got != tt.want {
			t.Errorf("Excluded(%q, %q) = %v, want %v", tt.path, tt.patterns, got, tt.want)
		}
	}
}

// makeTree creates files (with empty content) below a temporary directory and
// changes into it for the rest of the test.
func makeTree(t *testing.T, paths ...string) {
	t.Helper()
	root := t.TempDir()
	for _, p := range paths {
		full := filepath.Join(root, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("FROM scratch\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
}

func TestExpand(t *testing.T) {
	makeTree(t,
		"Dockerfile",
		"README.md",
		"services/api/Dockerfile",
		"services/api/Dockerfile.dockerignore",
		"services/web/web.Dockerfile",
		"services/web/node_modules/pkg/Dockerfile",
		".devcontainer/Dockerfile",
		".git/Dockerfile",
		"vendor/lib/Dockerfile",
		"notes/custom-name",
	)

	tests := []struct {
		name    string
		args    []string
		exclude []string
		want    []string
	}{
		{
			name: "directory",
			args: []string{"."},
			want: []string{".devcontainer/Dockerfile", "Dockerfile", "services/api/Dockerfile", "services/web/web.Dockerfile", "vendor/lib/Dockerfile"},
		},
		{
			name:    "directory with exclude",
			args:    []string{"."},
			exclude: []string{"vendor", ".devcontainer"},
			want:    []string{"Dockerfile", "services/api/Dockerfile", "services/web/web.Dockerfile"},
		},
		{
			name: "subdirectory",
			args: []string{"services"},
			want: []string{"services/api/Dockerfile", "services/web/web.Dockerfile"},
		},
		{
			name: "recursive pattern",
			args: []string{"**/Dockerfile"},
			want: []string{".devcontainer/Dockerfile", "Dockerfile", "services/api/Dockerfile", "vendor/lib/Dockerfile"},
		},
		{
			name: "pattern below a fixed directory",
			args: []string{"services/*/*"},
			want: []string{"services/api/Dockerfile", "services/api/Dockerfile.dockerignore", "services/web/web.Dockerfile"},
		},
		{
			name:    "explicit file is linted whatever its name and excludes",
			args:    []string{"notes/custom-name", "vendor/lib/Dockerfile"},
			exclude: []string{"vendor"},
			want:    []string{"notes/custom-name", "vendor/lib/Dockerfile"},
		},
		{
			name: "duplicates are dropped, order is kept",
			args: []string{"services/api/Dockerfile", "services", "Dockerfile"},
			want: []string{"services/api/Dockerfile", "services/web/web.Dockerfile", "Dockerfile"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Expand(tt.args, tt.exclude)
			if err != nil {
				t.Fatalf("Expand() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Expand(%q) =\n  %q\nwant\n  %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestExpandErrors(t *testing.T) {
	makeTree(t, "docs/README.md", "Dockerfile")

	tests := []struct {
		args []string
		want string
	}{
		{[]string{"missing/Dockerfile"}, "missing"},
		{[]string{"docs"}, "no Dockerfiles found"},
		{[]string{"**/*.Containerfile"}, "matched no files"},
		{[]string{"nope/**/Dockerfile"}, "matched no files"},
		{[]string{"Dockerfile", "[bad"}, "syntax error"},
	}
	for _, tt := range tests {
		_, err := Expand(tt.args, nil)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("Expand(%q) error = %v, want it to mention %q", tt.args, err, tt.want)
		}
	}
}
