package lang

import "strings"

type CommentForm struct {
	Line  string
	Block [2]string
}

type Language struct {
	Fence string `yaml:"fence"`
	Form  CommentForm
}

type Config struct {
	Languages map[string]Language `yaml:"languages"`
}

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

func Resolve(ext string, table map[string]Language) (Language, bool) {
	l, ok := table[strings.ToLower(strings.TrimPrefix(ext, "."))]
	return l, ok
}

func FenceFor(ext string, table map[string]Language) string {
	if l, ok := Resolve(ext, table); ok && l.Fence != "" {
		return l.Fence
	}
	return "text"
}
