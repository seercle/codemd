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

var unknownKeyPattern = regexp.MustCompile(`^line (\d+): field (.+) not found in type `)

func configDecodeError(path string, err error) error {
	var te *yaml.TypeError
	if !errors.As(err, &te) || len(te.Errors) == 0 {
		return fmt.Errorf("%s: %w", path, err)
	}
	lines := make([]string, 0, len(te.Errors))
	for _, entry := range te.Errors {
		m := unknownKeyPattern.FindStringSubmatch(entry)
		if m == nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		lines = append(lines, fmt.Sprintf("%s: line %s: unknown key %q", path, m[1], m[2]))
	}
	return errors.New(strings.Join(lines, "\n"))
}

// LoadConfig reads the YAML config at path and converts its language entries
// into a Config. It returns an error when the file cannot be read or parsed, or
// when an entry defines neither or both of line and block.
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
		var form CommentForm
		switch {
		case y.Line != "" && len(y.Block) > 0:
			return Config{}, fmt.Errorf("%s: language %q defines both line and block", path, ext)
		case y.Line != "":
			form.Line = y.Line
		case len(y.Block) == 2:
			form.Block = [2]string{y.Block[0], y.Block[1]}
		default:
			return Config{}, fmt.Errorf("%s: language %q must define exactly one of line or block", path, ext)
		}
		cfg.Languages[ext] = Language{Fence: y.Fence, Form: form}
	}
	return cfg, nil
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
// configured entry must define exactly one comment form; an entry without an
// explicit fence defaults to its extension. It returns an error for an invalid
// entry.
func Merge(base map[string]Language, cfg Config) (map[string]Language, error) {
	out := make(map[string]Language, len(base)+len(cfg.Languages))
	for k, v := range base {
		out[k] = v
	}
	for ext, l := range cfg.Languages {
		if l.Form.Line == "" && l.Form.Block == [2]string{} {
			return nil, fmt.Errorf("language %q must define exactly one of line or block", ext)
		}
		if l.Form.Line != "" && l.Form.Block != [2]string{} {
			return nil, fmt.Errorf("language %q defines both line and block", ext)
		}
		if l.Fence == "" {
			l.Fence = ext
		}
		out[ext] = l
	}
	return out, nil
}
