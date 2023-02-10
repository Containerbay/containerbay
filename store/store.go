package store

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/pterm/pterm"
)

// Store is a simple disk-cache store
// for container images
type Store struct {
	sync.Mutex
	dir string
}

// New returns a new store in the specified directory
func New(dir string) *Store {
	return &Store{dir: dir}
}

// EnsureExists ensures the store directory exists and have the correct permissions
func (s *Store) EnsureExists() error {
	return os.MkdirAll(s.dir, os.ModePerm)
}

// Exists checks weather the given key exists in the cache
func (s *Store) Exists(ss string) bool {
	if _, err := os.Stat(s.Path(ss)); err == nil {
		return true
	}
	return false
}

// Path is syntax sugar to return a subpath from the cache store
func (s *Store) Path(p ...string) string {
	return filepath.Join(append([]string{s.dir}, p...)...)
}

// String return the underlying directory store
func (s *Store) String() string {
	return s.dir
}

// CleanAll cleans up the cache store
func (s *Store) CleanAll() error {
	files, err := ioutil.ReadDir(s.dir)
	if err != nil {
		return err
	}

	for _, f := range files {
		os.RemoveAll(s.Path(f.Name()))
		pterm.Debug.Printfln("File '%s' pruned", f.Name())
	}
	return nil
}

// Clean cleans up the cache store only
// from the elements that aren't currently accessed
func (s *Store) Clean() error {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	files, err := ioutil.ReadDir(s.dir)
	if err != nil {
		return err
	}

	for _, f := range files {

		pterm.Debug.Printfln("File '%s' pruned", f.Name())
		os.RemoveAll(s.Path(f.Name()))
	}

	return nil
}
