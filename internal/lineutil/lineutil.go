package lineutil

import "strings"

type Lines struct {
	Content         []string
	EOL             string
	TrailingNewline bool
}

func Split(s string) Lines {
	l := Lines{EOL: "\n"}
	if s == "" {
		return l
	}
	l.TrailingNewline = strings.HasSuffix(s, "\n")
	body := s
	if l.TrailingNewline {
		body = s[:len(s)-1]
	}
	l.Content = strings.Split(body, "\n")
	for i, line := range l.Content {
		if strings.HasSuffix(line, "\r") {
			l.Content[i] = line[:len(line)-1]
			l.EOL = "\r\n"
		}
	}
	return l
}

func (l Lines) Join() string {
	if len(l.Content) == 0 {
		if l.TrailingNewline {
			return "\n"
		}
		return ""
	}
	out := strings.Join(l.Content, l.EOL)
	if l.TrailingNewline {
		out += l.EOL
	}
	return out
}
