// Package lang describes the source languages codemd understands and their
// comment syntaxes, and loads per-project overrides from a .codemd.yaml config.
package lang

import (
	"fmt"
	"sort"
	"strings"
)

// CommentForm describes a language's comment syntax: an optional Line prefix
// and/or a two-element Block pair holding the opening and closing delimiters.
type CommentForm struct {
	Line  string
	Block [2]string
}

// Language pairs the fenced-code info string used when emitting snippets with
// the language's comment syntax.
type Language struct {
	Fence string `yaml:"fence"`
	Form  CommentForm
}

// Config is the parsed form of a .codemd.yaml file: an extension-to-Language
// override map.
type Config struct {
	Languages map[string]Language `yaml:"languages"`
}

// Builtins returns the default extension-to-language table used when no
// .codemd.yaml overrides are supplied. Keys are bare file extensions without a
// leading dot.
func Builtins() map[string]Language {
	return map[string]Language{
		"go":   {Fence: "go", Form: CommentForm{Line: "//"}},
		"js":   {Fence: "javascript", Form: CommentForm{Line: "//"}},
		"mjs":  {Fence: "javascript", Form: CommentForm{Line: "//"}},
		"ts":   {Fence: "typescript", Form: CommentForm{Line: "//"}},
		"tsx":  {Fence: "tsx", Form: CommentForm{Line: "//"}},
		"jsx":  {Fence: "jsx", Form: CommentForm{Line: "//"}},
		"c":    {Fence: "c", Form: CommentForm{Line: "//", Block: [2]string{"/*", "*/"}}},
		"h":    {Fence: "c", Form: CommentForm{Line: "//", Block: [2]string{"/*", "*/"}}},
		"cpp":  {Fence: "cpp", Form: CommentForm{Line: "//", Block: [2]string{"/*", "*/"}}},
		"hpp":  {Fence: "cpp", Form: CommentForm{Line: "//", Block: [2]string{"/*", "*/"}}},
		"java": {Fence: "java", Form: CommentForm{Line: "//", Block: [2]string{"/*", "*/"}}},
		"rs":   {Fence: "rust", Form: CommentForm{Line: "//", Block: [2]string{"/*", "*/"}}},
		"py":   {Fence: "python", Form: CommentForm{Line: "#"}},
		"rb":   {Fence: "ruby", Form: CommentForm{Line: "#"}},
		"sh":   {Fence: "bash", Form: CommentForm{Line: "#"}},
		"bash": {Fence: "bash", Form: CommentForm{Line: "#"}},
		"yaml": {Fence: "yaml", Form: CommentForm{Line: "#"}},
		"yml":  {Fence: "yaml", Form: CommentForm{Line: "#"}},
		"toml": {Fence: "toml", Form: CommentForm{Line: "#"}},
		"lua":  {Fence: "lua", Form: CommentForm{Line: "--"}},
		"sql":  {Fence: "sql", Form: CommentForm{Line: "--"}},
		"html": {Fence: "html", Form: CommentForm{Block: [2]string{"<!--", "-->"}}},
		"xml":  {Fence: "xml", Form: CommentForm{Block: [2]string{"<!--", "-->"}}},
		"css":  {Fence: "css", Form: CommentForm{Block: [2]string{"/*", "*/"}}},
		"php":  {Fence: "php", Form: CommentForm{Line: "//", Block: [2]string{"/*", "*/"}}},
		"md":   {Fence: "markdown", Form: CommentForm{Block: [2]string{"<!--", "-->"}}},
	}
}

// Resolve looks up ext (case-insensitive, with an optional leading dot) in
// table and returns the matching Language. ok is false when the extension is
// unknown.
func Resolve(ext string, table map[string]Language) (Language, bool) {
	l, ok := table[strings.ToLower(strings.TrimPrefix(ext, "."))]
	return l, ok
}

// FenceFor returns the Markdown fenced-code info string for ext in table, or
// "text" when the extension is unknown or its entry has no fence.
func FenceFor(ext string, table map[string]Language) string {
	if l, ok := Resolve(ext, table); ok && l.Fence != "" {
		return l.Fence
	}
	return "text"
}

// Describe returns one human-readable line per language in table, sorted by
// extension. Each line is "ext -> fence (line "prefix")",
// "ext -> fence (block "open" "close")", or both joined with " or " for
// dual-form entries.
func Describe(table map[string]Language) []string {
	exts := make([]string, 0, len(table))
	for ext := range table {
		exts = append(exts, ext)
	}
	sort.Strings(exts)
	out := make([]string, 0, len(exts))
	for _, ext := range exts {
		l := table[ext]
		var parts []string
		if l.Form.Line != "" {
			parts = append(parts, fmt.Sprintf("line %q", l.Form.Line))
		}
		if l.Form.Block != [2]string{} {
			parts = append(parts, fmt.Sprintf("block %q %q", l.Form.Block[0], l.Form.Block[1]))
		}
		out = append(out, fmt.Sprintf("%s -> %s (%s)", ext, l.Fence, strings.Join(parts, " or ")))
	}
	return out
}
