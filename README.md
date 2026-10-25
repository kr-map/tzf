# TZF: a fast timezone finder for Go. [![Go Reference](https://pkg.go.dev/badge/github.com/Alfex4936/tzf.svg)](https://pkg.go.dev/github.com/Alfex4936/tzf) [![codecov](https://codecov.io/gh/Alfex4936/tzf/branch/main/graph/badge.svg?token=9KIU85IERM)](https://codecov.io/gh/Alfex4936/tzf)

![no_gc](https://github.com/Alfex4936/tzf/assets/2356749/ef5707b8-5639-4128-b7c9-c8658d82b9f3)
![gc](https://github.com/Alfex4936/tzf/assets/2356749/1af8374a-d0a8-4f32-b99f-4d2a0ce6cbd2)

TZF is a fast timezone finder package designed for Go. It allows you to quickly
find the timezone for a given latitude and longitude, making it ideal for geo
queries and services such as weather forecast APIs. With optimized performance
and two different data options, TZF is a powerful tool for any Go developer's
toolkit.

---

> [!NOTE]
>
> Here are some language or server which built with tzf or it's other language
> bindings:

| Language or Sever | Link                                                                    | Note              |
| ----------------- | ----------------------------------------------------------------------- | ----------------- |
| Go                | [`Alfex4936/tzf`](https://github.com/Alfex4936/tzf)                     |                   |
| Ruby              | [`HarlemSquirrel/tzf-rb`](https://github.com/HarlemSquirrel/tzf-rb)     |                   |
| Rust              | [`ringsaturn/tzf-rs`](https://github.com/ringsaturn/tzf-rs)             |                   |
| Python            | [`ringsaturn/tzfpy`](https://github.com/ringsaturn/tzfpy)               |                   |
| HTTP API          | [`ringsaturn/tzf-server`](https://github.com/ringsaturn/tzf-server)     | build with tzf    |
| HTTP API          | [`racemap/rust-tz-service`](https://github.com/racemap/rust-tz-service) | build with tzf-rs |
| Redis Server      | [`ringsaturn/tzf-server`](https://github.com/ringsaturn/tzf-server)     | build with tzf    |
| Redis Server      | [`ringsaturn/redizone`](https://github.com/ringsaturn/redizone)         | build with tzf-rs |

## Quick Start

To start using TZF in your Go project, you first need to install the package:

```bash
go get github.com/Alfex4936/tzf
```

Then, you can use the following code to locate:

```go
// Use about 150MB memory for init, and 60MB after GC.
package main

import (
	"fmt"

	"github.com/Alfex4936/tzf"
)

func main() {
	finder, err := tzf.NewDefaultFinder()
	if err != nil {
		panic(err)
	}
	fmt.Println(finder.GetTimezoneName(116.6386, 40.0786))
}
```

If you require a query result that is 100% accurate, use the following to
locate:

```go
// Use about 900MB memory for init, and 660MB after GC.
package main

import (
	"fmt"

	"github.com/Alfex4936/tzf"
	tzfrel "github.com/ringsaturn/tzf-rel"
	"github.com/Alfex4936/tzf/pb"
	"google.golang.org/protobuf/proto"
)

func main() {
	input := &pb.Timezones{}

	// Full data, about 83.5MB
	dataFile := tzfrel.FullData

	if err := proto.Unmarshal(dataFile, input); err != nil {
		panic(err)
	}
	finder, _ := tzf.NewFinderFromPB(input)
	fmt.Println(finder.GetTimezoneName(116.6386, 40.0786))
}
```

### Best Practice

It's expensive to init tzf's Finder/FuzzyFinder/DefaultFinder, please consider
reuse it or as a global var. Below is a global var example:

```go
package main

import (
	"fmt"

	"github.com/Alfex4936/tzf"
)

var f tzf.F

func init() {
	var err error
	f, err = tzf.NewDefaultFinder()
	if err != nil {
		panic(err)
	}
}

func main() {
	fmt.Println(f.GetTimezoneName(116.3883, 39.9289))
	fmt.Println(f.GetTimezoneName(-73.935242, 40.730610))
}
```

### Korea only

`korea.NewFinder()` embeds 20 KB of tzdata 2026e geometry (Asia/Seoul and its
neighbors, clipped to `korea.Bounds`) instead of the 6.7 MB world data. Inside
the box it answers like upstream's world finder (checked on 2.4M points by
`cmd/regiontzpb`); outside it returns `""`.

```go
f, err := korea.NewFinder()
inKorea := f.GetTimezoneName(126.9780, 37.5665) == "Asia/Seoul"
```

Apple M-series, `go test -bench . ./...`:

- Init: 0.9 ms and 1.4 MB allocated (DefaultFinder: 40 ms, 216 MB).
- Retained heap: 0.2 MB (DefaultFinder: 32 MB).
- Lookup: 2.5 ns inside the box (a 0.02° cell grid answers 93% of cells), about
  100-250 ns on borders, 2 ns outside (DefaultFinder: 70-1100 ns).

Regenerate after an upstream data release with `go generate ./korea`.

## CLI Tool

In addition to using TZF as a library in your Go projects, you can also use the
tzf command-line interface (CLI) tool to quickly get the timezone name for a set
of coordinates. To use the CLI tool, you first need to install it using the
following command:

```bash
go install github.com/Alfex4936/tzf/cmd/tzf@latest
```

Once installed, you can use the tzf command followed by the latitude and
longitude values to get the timezone name:

```bash
tzf -lng 116.3883 -lat 39.9289
```
