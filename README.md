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


## License

flagenv is licensed under a BSD-2-Clause license.
