package flagenv

import (
	"flag"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"
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

func cmp(w io.Writer, field string, a, b interface{}) bool {
	if field != "" {
		field = field + " = "
	}
	if reflect.DeepEqual(a, b) {
		_, _ = fmt.Fprintf(w, " %s%#+ v\n", field, b)
		return true
	}
	_, _ = fmt.Fprintf(w, "-%s%#+ v\n", field, a)
	_, _ = fmt.Fprintf(w, "+%s%#+ v\n", field, b)
	return false
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

	diff := func(a, b Flags) string {
		var out strings.Builder
		same := cmp(&out, "Int", a.Int, b.Int)
		same = same && cmp(&out, "Bool", a.Bool, b.Bool)
		same = same && cmp(&out, "Str", a.Str, b.Str)
		same = same && cmp(&out, "Strs", a.Strs, b.Strs)
		if same {
			return ""
		}
		return out.String()
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
				Strs: StringSlice{"1", "2"},
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

			if result := diff(c.Want, got); result != "" {
				t.Fatalf("Loaded values differ from expected values (-want +got):\n%s", result)
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

func TestCaseFuncs(t *testing.T) {
	type Case struct {
		Name string
		Fn   KeyFunc
		In   []string
		Want []string
	}

	cases := []Case{
		{
			Name: "SnakeCase",
			Fn:   SnakeCase,
			In: []string{
				"",
				"_______*****()",
				" Foo**Bar Baz___",
				"foo_bar-baz",
			},
			Want: []string{
				"",
				"_",
				"_Foo_Bar_Baz_",
				"foo_bar_baz",
			},
		},
		{
			Name: "DotCase",
			Fn:   DotCase,
			In: []string{
				"",
				"_______*****()",
				" Foo**Bar Baz___",
				"foo_bar-baz",
			},
			Want: []string{
				"",
				".",
				".Foo.Bar.Baz.",
				"foo.bar-baz",
			},
		},
		{
			Name: "KebabCase",
			Fn:   KebabCase,
			In: []string{
				"",
				"_______*****()",
				" Foo**Bar Baz___",
				"foo_bar-baz",
			},
			Want: []string{
				"",
				"-",
				"-Foo-Bar-Baz-",
				"foo-bar-baz",
			},
		},
	}

	diff := func(a, b string) string {
		var out strings.Builder
		if cmp(&out, "", a, b) {
			return ""
		}
		return out.String()
	}

	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			got := make([]string, len(c.In))
			for i, in := range c.In {
				got[i] = c.Fn(in)
			}
			for i := range c.Want {
				if result := diff(c.Want[i], got[i]); result != "" {
					t.Fatalf("%s conversion produced unexpected results (-want +got):\n%s", c.Name, result)
				}
			}
		})
	}
}
