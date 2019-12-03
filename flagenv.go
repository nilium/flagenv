package flagenv

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var errNoLookup = errors.New("no lookup function defined")

// LookupFunc is responsible for querying a config store and returning values for the given key.
// If a key doesn't exist, the LookupFunc should return an empty slice, not an error (unless a key
// missing really is an error).
type LookupFunc func(key string) ([]string, error)

// KeyFunc transforms a flag name into a key for use with a LookupFunc.
type KeyFunc func(name string) string

// Loader configures a flag loader to use particular Key and Lookup functions.
//
// The Lookup function may not be nil. If it is, it will return an error.
type Loader struct {
	Key    KeyFunc
	Lookup LookupFunc
}

func snakePrefix(str string) string {
	prefix := strings.ToUpper(SnakeCase(str)) + "_"
	if prefix[0] >= '0' && prefix[0] <= '9' {
		prefix = "_" + prefix
	}
	return prefix
}

// DefaultPrefix returns the default program prefix. This is an all-caps snake_case form of the
// program's filename.
func DefaultPrefix() string {
	return snakePrefix(filepath.Base(os.Args[0]))
}

// DefaultLoader returns a Loader with the default configuration for loading environment variables.
func DefaultLoader() *Loader {
	return EnvLoader(DefaultPrefix())
}

// EnvLoader returns a Loader that looks for keys with the given prefix. Keys are expected to be
// uppercased and separated by underscores. Values are loaded from environment variables.
func EnvLoader(prefix string) *Loader {
	return &Loader{
		Key:    Uppercased(WithPrefix(prefix, SnakeCase)),
		Lookup: WithIndexedLookup(LookupEnv, "_", 1),
	}
}

// DotLoader returns a Loader that looks for lowercased dot-separated keys with a given prefix.
// Keys can be defined with -1, -2, etc. suffixes if the key itself isn't defined as well.
// A lookup function must be provided.
func DotLoader(prefix string, lookup LookupFunc) *Loader {
	return &Loader{
		Key:    Lowercased(WithPrefix(prefix, DotCase)),
		Lookup: WithIndexedLookup(lookup, "-", 1),
	}
}

// defaultLoader is a static copy of the default flag loader.
var defaultLoader = DefaultLoader()

// SetAll sets all flags in the FlagSet.
//
// If a flag's value has a method `SkipMerge() bool` that returns true, then that flag is ignored
// by the Loader.
func (l *Loader) SetAll(f *flag.FlagSet) error {
	return l.setFlags(f, false)
}

// SetAll sets all flags in the FlagSet.
//
// This function uses the default loader.
func SetAll(f *flag.FlagSet) error {
	return defaultLoader.SetAll(f)
}

// SetOne sets the value of a single flag in the FlagSet.
// It returns an error if the flag doesn't exist.
func (l *Loader) SetOne(f *flag.FlagSet, name string) error {
	fv := f.Lookup(name)
	if fv == nil {
		return fmt.Errorf("flag not found: %s", name)
	}
	return l.setFlag(f, fv)
}

// SetOne sets the value of a single flag in the FlagSet.
// It returns an error if the flag doesn't exist.
//
// This function uses the default loader.
func SetOne(f *flag.FlagSet, name string) error {
	return defaultLoader.SetOne(f, name)
}

// SetMissing sets all flags that weren't already seen by the FlagSet.
//
// If a flag's value has a method `SkipMerge() bool` that returns true, then that flag is ignored
// by the Loader.
func (l *Loader) SetMissing(f *flag.FlagSet) error {
	return l.setFlags(f, true)
}

// SetMissing sets all flags that weren't already seen by the FlagSet.
//
// This function uses the default loader.
func SetMissing(f *flag.FlagSet) error {
	return defaultLoader.SetMissing(f)
}

func (l *Loader) setFlag(f *flag.FlagSet, fv *flag.Flag) error {
	if l.Lookup == nil {
		return errNoLookup
	}
	keyfn := l.Key
	if keyfn == nil {
		keyfn = Identity
	}
	name := fv.Name
	key := keyfn(name)
	if key == "" {
		return nil
	}
	values, err := l.Lookup(key)
	if err != nil {
		return fmt.Errorf("error looking up %s config with key %s: %w", name, key, err)
	}
	for _, value := range values {
		if err = f.Set(name, value); err != nil {
			return fmt.Errorf("unable to load %s config from key %s: %w", name, key, err)
		}
	}
	return nil
}

func (l *Loader) setFlags(f *flag.FlagSet, merge bool) (err error) {
	visited := func(*flag.Flag) bool { return false }
	if merge {
		seen := flagNames{}
		f.Visit(seen.visit)
		visited = seen.visited
	}

	f.VisitAll(func(fv *flag.Flag) {
		if err != nil || visited(fv) || shouldSkip(fv.Value) {
			return
		}
		if ferr := l.setFlag(f, fv); ferr != nil && err == nil {
			err = ferr
		}
	})

	return err
}

type mergeSkipper interface {
	SkipMerge() bool
}

func shouldSkip(v interface{}) bool {
	ms, ok := v.(mergeSkipper)
	return ok && ms.SkipMerge()
}

// flagNames is a simple string set for tracking visited flags in a flag.FlagSet.
type flagNames map[string]struct{}

func (fn flagNames) visit(f *flag.Flag) {
	fn[f.Name] = struct{}{}
}

func (fn flagNames) visited(f *flag.Flag) bool {
	_, ok := fn[f.Name]
	return ok
}
