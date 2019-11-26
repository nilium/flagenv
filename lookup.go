package flagenv

import (
	"math"
	"os"
	"strconv"
)

// LookupEnv looks up values for an environment variable of the given key.
func LookupEnv(key string) (values []string, err error) {
	v, ok := os.LookupEnv(key)
	if !ok {
		return nil, nil
	}
	return []string{v}, nil
}

// LookupMapValue returns a LookupFunc that looks up a single value in a map of strings to strings.
func LookupMapValue(m map[string]string) LookupFunc {
	return func(key string) ([]string, error) {
		v, ok := m[key]
		if !ok {
			return nil, nil
		}
		return []string{v}, nil
	}
}

// LookupMapValues returns a LookupFunc that looks up a slice of values in a map of strings to
// string slices.
func LookupMapValues(m map[string][]string) LookupFunc {
	return func(key string) ([]string, error) {
		v, ok := m[key]
		if !ok {
			return nil, nil
		}
		return v, nil
	}
}

type indexedLookup struct {
	innerLookup LookupFunc
	sep         string
	base        int
}

// WithIndexedLookup returns a LookupFunc that will look up indexed keys if the given key wasn't
// found. This can be useful for environment variables like STR_1, STR_2, STR_3, and so on.
//
// For example, to create a LookupFunc that would find STR_1 and onward:
//
//      lookup := flagenv.LookupEnv
//      lookup := flagenv.WithIndexedLookup(lookup, "_", 1)
//
// Then, if no value for STR is found, it will proceed to look for STR_1, and if that has a value,
// STR_2, and so on. If no value is found for an index, then the lookup stops.
func WithIndexedLookup(lookup LookupFunc, sep string, base int) LookupFunc {
	l := &indexedLookup{
		innerLookup: lookup,
		sep:         sep,
		base:        base,
	}
	return l.lookup
}

func (l *indexedLookup) lookup(key string) ([]string, error) {
	values, err := l.innerLookup(key)
	if err != nil {
		return nil, err
	}
	if len(values) != 0 {
		return values, nil
	}

	// Look up values of {KEY}{SEP}{INDEX} until one without a value is found.
	prefix := key + l.sep
	for i := l.base; i >= 0 && i < math.MaxInt32; i++ {
		ikey := prefix + strconv.Itoa(i)
		v, err := l.innerLookup(ikey)
		if err != nil {
			return nil, err
		}
		if len(v) == 0 {
			break
		}
		values = append(values, v...)
	}
	return values, nil
}
