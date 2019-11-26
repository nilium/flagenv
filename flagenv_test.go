package flagenv

import (
	"flag"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// Env is a map of strings to strings. It is intended for testing the LookupMapValue function.
type Env map[string]string

// StringSlice is a flag.Value implementation that appends strings to a slice.
type StringSlice []string

func (s *StringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func (s *StringSlice) String() string {
	return ""
}

func LogKeyAccess(t *testing.T, lookup LookupFunc) LookupFunc {
	return func(key string) ([]string, error) {
		v, err := lookup(key)
		t.Logf("Accessed key: %q (err = %v)", key, err)
		return v, err
	}
}

func TestLoader(t *testing.T) {
	simpleEnv := Env{
		// INI / property like:
		"Int":    "1234",
		"Bool":   "1",
		"Str":    "Hello!",
		"Strs.1": "Hello",
		"Strs.2": "World",
		"Strs.3": "!",
		"Strs.5": "Ignored",

		// Environment variable like:
		"APP_INT":  "256",
		"APP_BOOL": "0",
		"APP_STRS": "x",

		// Kebab-case:
		"app-int":    "256",
		"app-bool":   "0",
		"app-strs-0": "x",
		"app-strs-1": "y",
	}

	type Flags struct {
		Int  int
		Bool bool
		Str  string
		Strs StringSlice
	}

	type Case struct {
		Name   string
		SetFn  func(l *Loader, f *flag.FlagSet) error
		Loader Loader
		Args   []string
		Want   Flags
	}

	const (
		strVal  = "Hello!"
		intVal  = 1234
		boolVal = true
	)

	var strsVal = StringSlice{"Hello", "World", "!"}

	cases := []Case{
		// Dot case:
		{
			Name:  "Dot-NoFlagsPassed",
			SetFn: (*Loader).SetMissing,
			Loader: Loader{
				Key:    DotCase,
				Lookup: WithIndexedLookup(LookupMapValue(simpleEnv), ".", 1),
			},
			Want: Flags{
				Int:  intVal,
				Bool: true,
				Str:  strVal,
				Strs: strsVal,
			},
		},

		{
			Name: "Dot-AllFlagsPassed",
			Args: []string{
				"-Int=1",
				"-Bool=false",
				"-Str=Foobar",
				"-Strs=1",
				"-Strs=2",
			},
			SetFn: (*Loader).SetMissing,
			Loader: Loader{
				Key:    DotCase,
				Lookup: WithIndexedLookup(LookupMapValue(simpleEnv), ".", 1),
			},
			Want: Flags{
				Int:  1,
				Bool: false,
				Str:  "Foobar",
				Strs: StringSlice{"1", "2"},
			},
		},

		// Snake case:
		{
			Name: "Snake-MixedFlags",
			Args: []string{
				"-Bool",
			},
			SetFn: (*Loader).SetMissing,
			Loader: Loader{
				Key:    WithPrefix("APP_", Uppercased(SnakeCase)),
				Lookup: LookupMapValue(simpleEnv),
			},
			Want: Flags{
				Int:  256,
				Bool: true,
				Strs: StringSlice{"x"},
			},
		},

		{
			Name: "Snake-OverrideFlags",
			Args: []string{
				"-Int=1",
				"-Bool",
				"-Str=Foobar",
				"-Strs=1",
				"-Strs=2",
			},
			SetFn: (*Loader).SetAll,
			Loader: Loader{
				Key:    WithPrefix("APP_", Uppercased(SnakeCase)),
				Lookup: LookupMapValue(simpleEnv),
			},
			Want: Flags{
				Int:  256,
				Bool: false,
				Str:  "Foobar",
				Strs: StringSlice{"1", "2", "x"},
			},
		},

		// Kebab case:
		{
			Name: "Kebab-MixedFlags",
			Args: []string{
				"-Bool",
			},
			SetFn: (*Loader).SetMissing,
			Loader: Loader{
				Key:    WithPrefix("app-", Lowercased(KebabCase)),
				Lookup: WithIndexedLookup(LookupMapValue(simpleEnv), "-", 0),
			},
			Want: Flags{
				Int:  256,
				Bool: true,
				Strs: StringSlice{"x", "y"},
			},
		},

		{
			Name: "Kebab-OverrideFlags",
			Args: []string{
				"-Int=1",
				"-Bool",
				"-Str=Foobar",
				"-Strs=1",
				"-Strs=2",
			},
			SetFn: (*Loader).SetAll,
			Loader: Loader{
				Key:    WithPrefix("app-", Lowercased(KebabCase)),
				Lookup: WithIndexedLookup(LookupMapValue(simpleEnv), "-", 0),
			},
			Want: Flags{
				Int:  256,
				Bool: false,
				Str:  "Foobar",
				Strs: StringSlice{"1", "2", "x", "y"},
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()

			l := c.Loader
			l.Lookup = LogKeyAccess(t, l.Lookup)

			f := flag.NewFlagSet(c.Name, flag.PanicOnError)
			got := Flags{}

			f.IntVar(&got.Int, "Int", 0, "")
			f.BoolVar(&got.Bool, "Bool", false, "")
			f.StringVar(&got.Str, "Str", "", "")
			f.Var(&got.Strs, "Strs", "")

			_ = f.Parse(c.Args)

			if err := c.SetFn(&l, f); err != nil {
				t.Fatalf("Error setting flags: %v", err)
			}

			if diff := cmp.Diff(c.Want, got); diff != "" {
				t.Fatalf("Loaded values differ from expected values (-want +got):\n%s", diff)
			}
		})
	}

}

func TestSnakePrefix(t *testing.T) {
	type Case struct {
		In   string
		Want string
	}

	cases := []Case{
		{
			In:   "",
			Want: "_",
		},
		{
			In:   "foobar",
			Want: "FOOBAR_",
		},
		{
			In:   "9town",
			Want: "_9TOWN_",
		},
	}

	for _, c := range cases {
		got := snakePrefix(c.In)
		if got != c.Want {
			t.Fatalf("snakePrefix(%q) = %q; want %q", c.In, got, c.Want)
		}
	}
}
