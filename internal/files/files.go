// Package files expands the paths given on the command line into the list of
// Dockerfiles to lint.
package files

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// Expand turns each argument into Dockerfile paths:
//
//   - a file is linted as given, whatever its name;
//   - a directory is searched recursively for Dockerfiles (see IsDockerfile),
//     skipping .git and node_modules;
//   - a pattern with *, ? or [ is matched against every file below its fixed
//     leading directories, with ** matching any number of directories.
//
// Exclude patterns drop files found through a directory or a pattern; a file
// named explicitly is always linted. A directory or pattern that matches no
// file is an error, so a CI job cannot pass without linting anything.
//
// Paths found by expansion use forward slashes. The result has no duplicates
// and keeps the order of the arguments.
func Expand(args []string, exclude Exclusions) ([]string, error) {
	var result []string
	seen := make(map[string]bool)
	add := func(p string) {
		key := filepath.Clean(p)
		if !seen[key] {
			seen[key] = true
			result = append(result, p)
		}
	}

	for _, arg := range args {
		if hasMeta(arg) {
			matches, err := expandPattern(arg, exclude)
			if err != nil {
				return nil, err
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("%s: pattern matched no files", arg)
			}
			for _, m := range matches {
				add(m)
			}
			continue
		}

		info, err := os.Stat(arg)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			add(arg)
			continue
		}

		found, err := expandDir(arg, exclude)
		if err != nil {
			return nil, err
		}
		if len(found) == 0 {
			return nil, fmt.Errorf("%s: no Dockerfiles found in directory", arg)
		}
		for _, f := range found {
			add(f)
		}
	}
	return result, nil
}

// notDockerfileExtensions mark files named after a Dockerfile that are not one
// Docker can build: per-Dockerfile ignore files, templates rendered into a
// Dockerfile, editor and merge backups, and documentation.
var notDockerfileExtensions = []string{
	".dockerignore",
	".template", ".tmpl", ".tpl", ".j2", ".jinja", ".jinja2", ".erb", ".mustache", ".in",
	".bak", ".orig", ".old", ".swp", ".rej",
	".md", ".txt", ".sample", ".example",
}

// IsDockerfile reports whether a file name follows a Dockerfile naming
// convention: Dockerfile, Containerfile, Dockerfile.<suffix> or
// <prefix>.Dockerfile, in any letter case. Names ending in a template, backup
// or documentation extension (Dockerfile.template, Dockerfile.bak) are not
// Dockerfiles; directory searches skip them, so one unrendered template does
// not fail the whole run. Name such a file explicitly to lint it anyway.
func IsDockerfile(name string) bool {
	lower := strings.ToLower(name)
	for _, ext := range notDockerfileExtensions {
		if strings.HasSuffix(lower, ext) {
			return false
		}
	}
	for _, base := range []string{"dockerfile", "containerfile"} {
		if lower == base || strings.HasPrefix(lower, base+".") || strings.HasSuffix(lower, "."+base) {
			return true
		}
	}
	return false
}

func expandDir(dir string, exclude Exclusions) ([]string, error) {
	var found []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel := filepath.ToSlash(p)
		if d.IsDir() {
			if p != dir && skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if IsDockerfile(d.Name()) && !exclude.excludes(rel) {
			found = append(found, rel)
		}
		return nil
	})
	sort.Strings(found)
	return found, err
}

func expandPattern(pattern string, exclude Exclusions) ([]string, error) {
	pattern = filepath.ToSlash(pattern)
	if _, err := path.Match(strings.ReplaceAll(pattern, "**", "*"), ""); err != nil {
		return nil, fmt.Errorf("%s: %w", pattern, err)
	}

	root := patternRoot(pattern)
	var found []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && p == root {
				return filepath.SkipAll
			}
			return err
		}
		rel := filepath.ToSlash(p)
		if d.IsDir() {
			if p != root && skipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if Match(pattern, rel) && !exclude.excludes(rel) {
			found = append(found, rel)
		}
		return nil
	})
	sort.Strings(found)
	return found, err
}

// patternRoot returns the leading directories of a pattern that contain no
// wildcard, which is where the walk for matches starts.
func patternRoot(pattern string) string {
	segments := strings.Split(pattern, "/")
	var fixed []string
	for _, s := range segments[:len(segments)-1] {
		if hasMeta(s) {
			break
		}
		fixed = append(fixed, s)
	}
	if len(fixed) == 0 {
		return "."
	}
	root := strings.Join(fixed, "/")
	if root == "" {
		return "/" // absolute pattern such as /*/Dockerfile
	}
	return root
}

// skipDir reports whether a directory is skipped while searching: the Git
// metadata and installed JavaScript packages, which hold no project
// Dockerfiles. Other hidden directories such as .devcontainer are searched.
func skipDir(name string) bool {
	return name == ".git" || name == "node_modules"
}

// Exclusions are exclude patterns and the directory they are written relative
// to: the directory of the configuration file that declares them. Matching
// against paths made relative to Base keeps a pattern such as "examples/**"
// working when docker-lint runs from a subdirectory or is given absolute paths.
type Exclusions struct {
	Patterns []string
	// Base is the directory patterns are relative to; "" means the current one.
	Base string
}

func (e Exclusions) excludes(p string) bool {
	if len(e.Patterns) == 0 {
		return false
	}
	return Excluded(relativeTo(e.Base, p), e.Patterns)
}

// relativeTo returns p relative to base with forward slashes, or p unchanged
// when it lies outside base.
func relativeTo(base, p string) string {
	if base == "" {
		base = "."
	}
	absBase, errBase := filepath.Abs(base)
	absPath, errPath := filepath.Abs(p)
	if errBase == nil && errPath == nil {
		rel, err := filepath.Rel(absBase, absPath)
		if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return filepath.ToSlash(rel)
		}
	}
	return filepath.ToSlash(p)
}

// Excluded reports whether a slash-separated path matches any exclude pattern.
// A pattern without a slash matches a file or directory name anywhere in the
// path, as in .gitignore; other patterns match the path from its start, and a
// directory pattern also excludes everything inside it.
func Excluded(p string, patterns []string) bool {
	p = strings.TrimPrefix(path.Clean(p), "./")
	segments := strings.Split(p, "/")
	for _, pattern := range patterns {
		pattern = strings.TrimSuffix(strings.TrimPrefix(filepath.ToSlash(pattern), "./"), "/")
		if pattern == "" {
			continue
		}
		if !strings.Contains(pattern, "/") {
			for _, s := range segments {
				if ok, _ := path.Match(pattern, s); ok {
					return true
				}
			}
			continue
		}
		// Match the pattern against the path and each of its parent directories.
		for i := len(segments); i > 0; i-- {
			if Match(pattern, strings.Join(segments[:i], "/")) {
				return true
			}
		}
	}
	return false
}

// Match reports whether a slash-separated path matches a pattern in which each
// segment follows path.Match syntax and a ** segment matches zero or more
// directories. A leading "./" on either side is ignored.
func Match(pattern, p string) bool {
	pattern = strings.TrimPrefix(pattern, "./")
	p = strings.TrimPrefix(path.Clean(p), "./")
	return matchSegments(strings.Split(pattern, "/"), strings.Split(p, "/"))
}

func matchSegments(pattern, segments []string) bool {
	for len(pattern) > 0 {
		if pattern[0] == "**" {
			// Collapse repeated ** and try every possible split point.
			rest := pattern[1:]
			for len(rest) > 0 && rest[0] == "**" {
				rest = rest[1:]
			}
			if len(rest) == 0 {
				return true
			}
			for i := 0; i <= len(segments); i++ {
				if matchSegments(rest, segments[i:]) {
					return true
				}
			}
			return false
		}
		if len(segments) == 0 {
			return false
		}
		if ok, err := path.Match(pattern[0], segments[0]); err != nil || !ok {
			return false
		}
		pattern, segments = pattern[1:], segments[1:]
	}
	return len(segments) == 0
}

func hasMeta(s string) bool {
	return strings.ContainsAny(s, "*?[")
}
