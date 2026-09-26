package srcfile

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Loader struct {
	Client *http.Client
	cache  map[string]string
}

func NewLoader() *Loader {
	return &Loader{
		Client: &http.Client{Timeout: 30 * time.Second},
		cache:  map[string]string{},
	}
}

func (l *Loader) Load(path, baseDir string) (string, bool, error) {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		if cached, ok := l.cache[path]; ok {
			return cached, true, nil
		}
		resp, err := l.Client.Get(path)
		if err != nil {
			return "", true, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return "", true, fmt.Errorf("%s: HTTP %d", path, resp.StatusCode)
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", true, err
		}
		l.cache[path] = string(data)
		return string(data), true, nil
	}
	resolved := path
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(baseDir, resolved)
	}
	if cached, ok := l.cache[resolved]; ok {
		return cached, false, nil
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return "", false, err
	}
	l.cache[resolved] = string(data)
	return string(data), false, nil
}
