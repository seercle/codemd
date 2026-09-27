// Package srcfile loads local and remote source files and extracts codemd
// markers from their comments.
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

// Loader reads local and http(s) source files, caching their contents. Build
// one with NewLoader; the zero value is not ready for use.
type Loader struct {
	Client *http.Client
	cache  map[string]string
}

// NewLoader returns a Loader with a 30-second-timeout HTTP client and an empty
// cache.
func NewLoader() *Loader {
	return &Loader{
		Client: &http.Client{Timeout: 30 * time.Second},
		cache:  map[string]string{},
	}
}

// Load returns the contents of path, resolved relative to baseDir unless it is
// absolute or an http(s) URL. The boolean reports whether the source was
// remote. Results are memoised in the Loader's cache.
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
