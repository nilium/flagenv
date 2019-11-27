# flagenv

[![GoDoc](https://godoc.org/go.spiff.io/flagenv?status.svg)](https://godoc.org/go.spiff.io/flagenv)

```go
import "go.spiff.io/flagenv"
```


Flagenv is a simple package to load environment variables for arguments set
using the [flag][] package's [FlagSet][] type.

The purpose of this package is to keep a very small footprint for loading CLI
and environment-based configuration.

[flag]: https://golang.org/pkg/flag/
[FlagSet]: https://golang.org/pkg/flag/#FlagSet


## Usage

For a simple program named `example`, the following will load any CLI flags not
passed on the command line from environment variables by using the
[SetMissing][] function:

```go
package main // github.com/yourname/yourprogram/example

import (
        "flag"
        "log"

        "go.spiff.io/flagenv"
)

func main() {
        workers := flag.Int("worker-processes", 8, "The number of worker subprocesses to spawn.")
        user := flag.String("worker-user", "nobody", "The user to run the worker as.")
        group := flag.String("worker-group", "nogroup", "The group to run the worker as.")
        flag.Parse()

        // Load values from environment variables.
        if err := flagenv.SetMissing(flag.CommandLine); err != nil {
                log.Fatalf("Error configuring example: %v", err)
        }

        log.Printf("Workers=%d User=%q Group=%q", *workers, *user, *group)
}
```

This can then be run with the following:

```
$ go build github.com/yourname/yourprogram/example

$ EXAMPLE_WORKER_PROCESSES=32000 ./example -worker-user shodan
2006/01/02 15:04:05 Workers=32000 User="shodan" Group="nogroup"
```

Additional customization can be done by combining different [KeyFuncs][KeyFunc] and
[LookupFuncs][LookupFunc] when creating a [Loader][].

[Loader]: https://godoc.org/go.spiff.io/flagenv#Loader
[KeyFunc]: https://godoc.org/go.spiff.io/flagenv#KeyFunc
[LookupFunc]: https://godoc.org/go.spiff.io/flagenv#LookupFunc
[SetMissing]: https://godoc.org/go.spiff.io/flagenv#SetMissing


## Load Values From INI Files

It's possible to use flagenv to load configuration from INI files using
[go.spiff.io/go-ini][] (or a similar library) and a flag.Value implementation. For
example:

[go.spiff.io/go-ini]: https://godoc.org/go.spiff.io/go-ini

```go
package main // github.com/yourname/yourprogram/example2

import (
        "bufio"
        "flag"
        "fmt"
        "log"
        "os"

        "go.spiff.io/flagenv"
        ini "go.spiff.io/go-ini"
)

func main() {
        // Add a -config flag that allows loading a config file during CLI parsing.
        flag.Var(Configger{flag.CommandLine}, "config", "Load a config file.")
        workers := flag.Int("worker-processes", 8, "The number of worker subprocesses to spawn.")
        user := flag.String("worker-user", "nobody", "The user to run the worker as.")
        group := flag.String("worker-group", "nogroup", "The group to run the worker as.")
        flag.Parse()

        // Load values from environment variables.
        if err := flagenv.SetMissing(flag.CommandLine); err != nil {
                log.Fatalf("Error configuring example: %v", err)
        }

        log.Printf("Workers=%d User=%q Group=%q", *workers, *user, *group)
}

// Configger loads config files and assigns their values to a destination FlagSet.
type Configger struct {
        dst *flag.FlagSet
}

func (c Configger) readFile(dst ini.Recorder, file string) error {
        f, err := os.Open(file)
        if err != nil {
                return err
        }
        defer f.Close()
        r := ini.Reader{True: "true"}
        return r.Read(bufio.NewReader(f), dst)
}

// Set loads an INI file and assigns values frm it to the destination FlagSet.
func (c Configger) Set(file string) error {
        values := map[string][]string{}
        if err := c.readFile(ini.Values(values), file); err != nil {
                return fmt.Errorf("unable to load config file %s: %w", file, err)
        }
        iniLoader := flagenv.Loader{
                Key:    flagenv.WithPrefix("example.", flagenv.Lowercased(flagenv.DotCase)),
                Lookup: flagenv.LookupMapValues(values),
        }
        // Use SetAll to assign all values from the config file. This overrides previously-seen CLI
        // arguments, but CLI arguments following after the config file will still take effect.
        if err := iniLoader.SetAll(c.dst); err != nil {
                return fmt.Errorf("error loading config file %s: %w", file, err)
        }
        return nil
}

func (c Configger) String() string { return "" }
```

You can then run it with the following:

```
$ go build github.com/yourname/yourprogram/example2

$ ./example2 -config /dev/stdin -worker-user not-shodan <<INI
[example]
worker-user = shodan
worker-processes = 1234
INI
2006/01/02 15:04:05 Workers=1234 User="not-shodan" Group="nogroup"
```

This works by loading the config file when the -config flag is parsed. By doing
this, it allows us to use config files to override previously-set flag values
and, at the same time, for subsequent flags to override values from the config
file. Meanwhile, using SetMissing with the default Loader afterward lets us fill
in gaps in configuration with environment variables.


## License

flagenv is licensed under a BSD-2-Clause license.
