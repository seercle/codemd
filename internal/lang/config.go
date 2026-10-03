package lang

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type yamlLanguage struct {
	Line  string   `yaml:"line"`
	Block []string `yaml:"block"`
	Fence string   `yaml:"fence"`
}

var (
	unknownKeyPattern = regexp.MustCompile(`^line (\d+): field (.+) not found in type `)
	goTypePattern     = regexp.MustCompile(` (?:in type|into) \S+`)
)

// configDecodeError rewrites yaml.v3's strict-decode errors so they name the
// config file and the offending key without exposing Go type names.
func configDecodeError(path string, err error) error {
	var te *yaml.TypeError
	if !errors.As(err, &te) || len(te.Errors) == 0 {
		return fmt.Errorf("%s: %w", path, err)
	}
	lines := make([]string, 0, len(te.Errors))
	for _, entry := range te.Errors {
		if m := unknownKeyPattern.FindStringSubmatch(entry); m != nil {
			lines = append(lines, fmt.Sprintf("%s: line %s: unknown key %q", path, m[1], m[2]))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s: %s", path, goTypePattern.ReplaceAllString(entry, "")))
	}
	return errors.New(strings.Join(lines, "\n"))
}

// LoadConfig reads the YAML config at path and converts its language entries
// into a Config. It returns an error when the file cannot be read or parsed,
// when the document has unknown keys, or when an entry defines neither line nor
// block, or a block that is not exactly two elements.
func LoadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var raw struct {
		Languages map[string]yamlLanguage `yaml:"languages"`
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&raw); err != nil && !errors.Is(err, io.EOF) {
		return Config{}, configDecodeError(path, err)
	}
	cfg := Config{Languages: map[string]Language{}}
	for ext, y := range raw.Languages {
		if len(y.Block) != 0 && len(y.Block) != 2 {
			return Config{}, fmt.Errorf("%s: language %q block must have exactly two elements", path, ext)
		}
		var form CommentForm
		form.Line = y.Line
		if len(y.Block) == 2 {
			form.Block = [2]string{y.Block[0], y.Block[1]}
		}
		if err := validateForm(ext, form); err != nil {
			return Config{}, fmt.Errorf("%s: %w", path, err)
		}
		cfg.Languages[ext] = Language{Fence: y.Fence, Form: form}
	}
	return cfg, nil
}

// validateForm reports whether a language entry defines at least one comment
// form.
func validateForm(ext string, form CommentForm) error {
	if form.Line == "" && form.Block == [2]string{} {
		return fmt.Errorf("language %q must define at least one of line or block", ext)
	}
	return nil
}

// DiscoverConfig walks up from startDir looking for a .codemd.yaml file. It
// returns the path of the first match, or "" with a nil error when none is
// found up to the filesystem root.
func DiscoverConfig(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, ".codemd.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// Merge returns a copy of base with cfg's language entries overlaid. Each
// configured entry must define at least one comment form; an entry without an
// explicit fence defaults to its extension. It returns an error for an invalid
// entry.
func Merge(base map[string]Language, cfg Config) (map[string]Language, error) {
	out := make(map[string]Language, len(base)+len(cfg.Languages))
	for k, v := range base {
		out[k] = v
	}
	for ext, l := range cfg.Languages {
		if err := validateForm(ext, l.Form); err != nil {
			return nil, err
		}
		if l.Fence == "" {
			l.Fence = ext
		}
		out[ext] = l
	}
	return out, nil
}
