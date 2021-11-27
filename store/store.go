package store

import (
	"fmt"
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
	l   map[string]*sync.Mutex
}

// New returns a new store in the specified directory
func New(dir string) *Store {
	return &Store{dir: dir, l: make(map[string]*sync.Mutex)}
}

// EnsureExists ensures the store directory exists and have the correct permissions
func (s *Store) EnsureExists() error {
	return os.MkdirAll(s.dir, os.ModePerm)
}

// Exists checks weather the given key exists in the cache
func (s *Store) Exists(ss string) bool {
	if _, err := os.Stat(filepath.Join(s.dir, ss)); err == nil {
		return true
	}
	return false
}

// Path is syntax sugar to return a subpath from the cache store
func (s *Store) Path(p ...string) string {
	return filepath.Join(append([]string{s.dir}, p...)...)
}

func (s *Store) lockPath(p string) string {
	return s.Path(fmt.Sprintf("%s.lock", p))
}

// String return the underlying directory store
func (s *Store) String() string {
	return s.dir
}

// Lock locks the access store for the given cache key
// It is blocking
func (s *Store) Lock(p string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()

	if l, exists := s.l[p]; exists {
		l.Lock()
	} else {
		s.l[p] = &sync.Mutex{}
		s.l[p].Lock()
	}

	pterm.Debug.Println("Acquiring lock", p)
}

// Unlock unlocks the access store for the given cache key
func (s *Store) Unlock(p string) {
	s.Mutex.Lock()
	defer s.Mutex.Unlock()
	if l, exists := s.l[p]; exists {
		l.Unlock()
		delete(s.l, p)
	}

	pterm.Debug.Println("Unlocked", p)
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

		if _, ok := s.l[f.Name()]; ok {
			pterm.Debug.Printfln("file %s locked", f.Name())
			continue
		}

		pterm.Debug.Printfln("File '%s' pruned", f.Name())
		os.RemoveAll(s.Path(f.Name()))
	}

	return nil
}
