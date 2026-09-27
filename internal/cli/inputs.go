package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const globMeta = "*?["

func hasGlobMeta(s string) bool {
	return strings.ContainsAny(s, globMeta)
}

// expandInputs turns positional arguments into a de-duplicated list of files.
// Directories are walked recursively for Markdown files; arguments containing
// glob metacharacters are expanded (with ** support); an explicit file is kept
// as-is. An argument that matches nothing is an error.
func expandInputs(args []string) ([]string, error) {
	var out []string
	seen := map[string]bool{}
	add := func(p string) {
		clean := filepath.Clean(p)
		if seen[clean] {
			return
		}
		seen[clean] = true
		out = append(out, clean)
	}
	for _, arg := range args {
		if hasGlobMeta(arg) {
			matches, err := globExpand(arg)
			if err != nil {
				return nil, err
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("no files match %q", arg)
			}
			for _, m := range matches {
				add(m)
			}
			continue
		}
		info, err := os.Stat(arg)
		switch {
		case err == nil && info.IsDir():
			files, werr := walkMarkdown(arg)
			if werr != nil {
				return nil, werr
			}
			if len(files) == 0 {
				return nil, fmt.Errorf("no Markdown files in %q", arg)
			}
			for _, f := range files {
				add(f)
			}
		case err == nil:
			add(arg)
		default:
			return nil, fmt.Errorf("%s: %w", arg, err)
		}
	}
	return out, nil
}

// walkMarkdown returns the Markdown files under dir (recursively), skipping
// directories whose name starts with '.'.
func walkMarkdown(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != dir && strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".md", ".markdown":
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// globExpand expands a glob pattern, supporting "**" for any number of path
// segments. Only regular files are returned.
func globExpand(pattern string) ([]string, error) {
	var matches []string
	err := filepath.WalkDir(globRoot(pattern), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if matchSegments(pattern, path) {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(matches)
	return matches, nil
}

// globRoot is the longest leading directory of pattern that contains no glob
// metacharacters. It is the directory globExpand walks.
func globRoot(pattern string) string {
	slashed := filepath.ToSlash(pattern)
	parts := strings.Split(slashed, "/")
	root := "."
	if strings.HasPrefix(slashed, "/") {
		root = "/"
	}
	for i, p := range parts {
		if i == 0 && p == "" {
			continue
		}
		if i == len(parts)-1 || hasGlobMeta(p) {
			break
		}
		root = filepath.Join(root, p)
	}
	return root
}

// matchSegments reports whether path matches pattern, where "**" matches zero
// or more whole path segments and other segments use filepath.Match semantics.
func matchSegments(pattern, path string) bool {
	pat := strings.Split(filepath.ToSlash(pattern), "/")
	segs := strings.Split(filepath.ToSlash(path), "/")
	return matchParts(pat, segs)
}

func matchParts(pat, segs []string) bool {
	for len(pat) > 0 {
		if pat[0] == "**" {
			if matchParts(pat[1:], segs) {
				return true
			}
			if len(segs) == 0 {
				return false
			}
			segs = segs[1:]
			continue
		}
		if len(segs) == 0 {
			return false
		}
		ok, err := filepath.Match(pat[0], segs[0])
		if err != nil || !ok {
			return false
		}
		pat = pat[1:]
		segs = segs[1:]
	}
	return len(segs) == 0
}
