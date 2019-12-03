package flagenv

import (
	"errors"
	"flag"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// ignoreKeys is a test KeyFunc to ensure that empty keys aren't looked up.
func ignoreKeys(keyfn KeyFunc, keys ...string) KeyFunc {
	strs := map[string]struct{}{}
	for _, k := range keys {
		strs[k] = struct{}{}
	}
	return func(key string) string {
		if _, ok := strs[key]; ok {
			return ""
		}
		return keyfn(key)
	}
}

func setLookupEnv(fn func(string) (string, bool)) func() {
	last := osLookupEnv
	osLookupEnv = fn
	return func() {
		osLookupEnv = last
	}
}

// Env is a map of strings to strings. It is intended for testing the LookupMapValue function.
type Env map[string]string

// ValuesEnv is a map of strings to string slices. It is indented for testing the LookupMapValues
// function.
type ValuesEnv map[string][]string

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
		t.Logf("Accessed key: %q value = %q (err = %v)", key, v, err)
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

		// Dot loader:
		"app.int":    "256",
		"app.bool":   "0",
		"app.strs-1": "x",
		"app.strs-2": "y",

		// Kebab-case:
		"app-int":    "256",
		"app-bool":   "0",
		"app-strs-0": "x",
		"app-strs-1": "y",
	}

	valuesEnv := ValuesEnv{
		"Bool": {"true", "false"},
		"Strs": {"1", "2", "3"},
		"Int":  {"1", "2", "3", "450"},
	}

	// Default prefix:
	defaultPrefix := DefaultPrefix()
	defaultEnv := Env{
		defaultPrefix + "INT":    "940",
		defaultPrefix + "BOOL":   "true",
		defaultPrefix + "STRS_1": "left",
		defaultPrefix + "STRS_2": "center",
		defaultPrefix + "STRS_3": "right",
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
		Loader *Loader
		Args   []string
		Want   Flags
	}

	const (
		strVal  = "Hello!"
		intVal  = 1234
		boolVal = true
	)

	var strsVal = StringSlice{"Hello", "World", "!"}

	defer setLookupEnv(func(key string) (string, bool) {
		v, ok := defaultEnv[key]
		return v, ok
	})()

	cases := []Case{
		// Dot case:
		{
			Name:  "Dot-NoFlagsPassed",
			SetFn: (*Loader).SetMissing,
			Loader: &Loader{
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
			Loader: &Loader{
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

		// DotLoader:
		{
			Name: "DotLoader",
			Args: []string{
				"-Bool",
			},
			SetFn:  (*Loader).SetMissing,
			Loader: DotLoader("app.", LookupMapValue(simpleEnv)),
			Want: Flags{
				Int:  256,
				Bool: true,
				Strs: StringSlice{"x", "y"},
			},
		},

		// Snake case:
		{
			Name: "Snake-MixedFlags",
			Args: []string{
				"-Bool",
			},
			SetFn: (*Loader).SetMissing,
			Loader: &Loader{
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
			Loader: &Loader{
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
			Loader: &Loader{
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
			Loader: &Loader{
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

		// Empty keys are ignored:
		{
			Name: "EmptyKeys",
			Args: []string{
				"-Int=128",
				"-Str=CLI",
				"-Bool=true",
			},
			SetFn: (*Loader).SetAll,
			Loader: &Loader{
				Key:    WithPrefix("app-", Lowercased(ignoreKeys(KebabCase, "Str", "Bool", "Strs"))),
				Lookup: WithIndexedLookup(LookupMapValue(simpleEnv), "-", 0),
			},
			Want: Flags{
				Int:  256,
				Bool: true,
				Str:  "CLI",
			},
		},

		// LookupMapValues:
		{
			Name:  "LookupMapValues",
			SetFn: (*Loader).SetMissing,
			Loader: &Loader{
				Lookup: LookupMapValues(valuesEnv),
			},
			Want: Flags{
				Int:  450,
				Bool: false,
				Strs: StringSlice{"1", "2", "3"},
			},
		},

		// Default loader:
		{
			Name: "DefaultLoader-SetMissing",
			Args: []string{
				"-Int=1",
				"-Bool=false",
				"-Str=Foobar",
			},
			SetFn:  func(_ *Loader, f *flag.FlagSet) error { return SetMissing(f) },
			Loader: DefaultLoader(),
			Want: Flags{
				Int:  1,
				Bool: false,
				Str:  "Foobar",
				Strs: StringSlice{"left", "center", "right"},
			},
		},

		{
			Name: "DefaultLoader-SetOne",
			Args: []string{
				"-Int=1",
				"-Bool=false",
				"-Str=Foobar",
			},
			SetFn:  func(_ *Loader, f *flag.FlagSet) error { return SetOne(f, "Int") },
			Loader: DefaultLoader(),
			Want: Flags{
				Int:  940,
				Bool: false,
				Str:  "Foobar",
			},
		},

		{
			Name: "DefaultLoader-SetAll",
			Args: []string{
				"-Int=1",
				"-Bool=false",
				"-Str=Foobar",
			},
			SetFn:  func(_ *Loader, f *flag.FlagSet) error { return SetAll(f) },
			Loader: DefaultLoader(),
			Want: Flags{
				Int:  940,
				Bool: true,
				Str:  "Foobar",
				Strs: StringSlice{"left", "center", "right"},
			},
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			l := c.Loader
			l.Lookup = LogKeyAccess(t, l.Lookup)

			f := flag.NewFlagSet(c.Name, flag.PanicOnError)
			got := Flags{}

			f.IntVar(&got.Int, "Int", 0, "")
			f.BoolVar(&got.Bool, "Bool", false, "")
			f.StringVar(&got.Str, "Str", "", "")
			f.Var(&got.Strs, "Strs", "")

			_ = f.Parse(c.Args)

			if err := c.SetFn(l, f); err != nil {
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

	for _, c := range cases {
		c := c
		t.Run(c.Name, func(t *testing.T) {
			got := make([]string, len(c.In))
			for i, in := range c.In {
				got[i] = c.Fn(in)
			}
			if diff := cmp.Diff(c.Want, got); diff != "" {
				t.Fatalf("%s conversion produced unexpected results (-want +got):\n%s", c.Name, diff)
			}
		})
	}
}

func TestLookupError(t *testing.T) {
	f := flag.NewFlagSet("test", flag.ContinueOnError)
	_ = f.Int("Int", 0, "")

	ld := &Loader{}
	if err := ld.SetAll(f); !errors.Is(err, errNoLookup) {
		t.Errorf("Loader without Lookup returned %#v; want %#v", err, errNoLookup)
	}

	// Lookup failure:
	want := errors.New("test error")
	ld.Lookup = func(key string) ([]string, error) {
		return nil, want
	}
	if err := ld.SetAll(f); !errors.Is(err, want) {
		t.Errorf("Lookup returned %#v; want %#v", err, want)
	}

	// Indexed lookup failure:
	ld.Lookup = WithIndexedLookup(ld.Lookup, "_", 1)
	if err := ld.SetAll(f); !errors.Is(err, want) {
		t.Errorf("Indexed Lookup returned %#v; want %#v", err, want)
	}

	// Indexed lookup failure, after initial lookup:
	ld.Lookup = WithIndexedLookup(func(key string) ([]string, error) {
		if strings.HasSuffix(key, "_1") {
			return nil, want
		}
		return nil, nil
	}, "_", 1)
	if err := ld.SetAll(f); !errors.Is(err, want) {
		t.Errorf("Indexed Lookup returned %#v; want %#v", err, want)
	}

	// FlagSet.Set failure:
	ld.Lookup = func(key string) ([]string, error) {
		return []string{"not-an-int"}, nil
	}
	if err := ld.SetAll(f); err == nil {
		t.Error("Set did not return a parse error when one was expected")
	}

	// SetOne should return an error when a flag doesn't exist.
	if err := ld.SetOne(f, "NotAFlag"); err == nil {
		t.Error("SetOne did not return an error for an undefined flag when one was expected")
	}
}
