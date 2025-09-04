# gowild

[![Go Reference](https://pkg.go.dev/badge/github.com/twinfer/gowild.svg)](https://pkg.go.dev/github.com/twinfer/gowild)
[![Go Report Card](https://goreportcard.com/badge/github.com/twinfer/gowild)](https://goreportcard.com/report/github.com/twinfer/gowild)

`gowild` is a lightweight and highly optimized Go package for matching strings against patterns containing wildcards. It is designed for performance-critical applications that require fast pattern matching on ASCII or byte-oriented strings.

## Features

- **Fast:** Includes performance optimizations for common patterns to avoid unnecessary allocations.
- **Flexible Wildcards:** Supports `*`, `?`, and `.` wildcards.
- **Unicode Support:** Provides separate functions for correct, rune-wise matching of Unicode strings.
- **Case-Insensitive Matching:** Includes functions for case-insensitive comparisons.
- **Byte-slice Support:** Provides functions that operate directly on `[]byte` to reduce allocations.

### Supported Wildcards

- `*` - Matches any sequence of characters (including zero characters).
- `?` - Matches any single character or zero characters (an optional character).
- `.` - Matches any single character (the character must be present).

## Installation

```sh
go get github.com/twinfer/gowild
```

## Usage

### Basic Matching

The `gowild.Match` function is the fastest option, ideal for ASCII strings.

```go
package main

import (
	"fmt"
	"github.com/twinfer/gowild"
)

func main() {
	// Simple match
	match := gowild.Match("h?llo*world", "hello beautiful world")
	fmt.Println(match) // Output: true

	// Does not match
	match = gowild.Match("h?llo*world", "goodbye world")
	fmt.Println(match) // Output: false
}
```

### Unicode Matching

When working with strings that may contain multi-byte Unicode characters, use `gowild.MatchByRune`.

```go
package main

import (
	"fmt"
	"github.com/twinfer/gowild"
)

func main() {
	// The `.` wildcard correctly matches the multi-byte 'é' character.
	match := gowild.MatchByRune("caf.", "café")
	fmt.Println(match) // Output: true

	// This would fail with the standard gowild.Match function.
	match = gowild.Match("caf.", "café")
	fmt.Println(match) // Output: false
}
```

### Case-Insensitive Matching

Use `gowild.MatchFold` for case-insensitive matching of ASCII strings.

```go
package main

import (
	"fmt"
	"github.com/twinfer/gowild"
)

func main() {
	match := gowild.MatchFold("HELLO*.?", "hello world")
	fmt.Println(match) // Output: true
}
```

## API Overview

| Function        | Case-Sensitive | Unicode-Aware | Input Type | Description                                                              |
|-----------------|----------------|---------------|------------|--------------------------------------------------------------------------|
| `Match`         | Yes            | No (Bytes)    | `string`   | Fastest option, for ASCII or byte-oriented matching.                     |
| `MatchByRune`   | Yes            | Yes           | `string`   | For correct matching of strings with Unicode characters.                 |
| `MatchFromByte` | Yes            | No (Bytes)    | `[]byte`   | Byte-slice version of `Match` to avoid allocations.                      |
| `MatchFold`     | No             | No (Bytes)    | `string`   | Case-insensitive version of `Match`.                                     |
| `MatchFoldRune` | No             | Yes           | `string`   | Case-insensitive version of `MatchByRune`. The most "correct" version. |
| `MatchFoldByte` | No             | No (Bytes)    | `[]byte`   | Byte-slice version of `MatchFold`.                                       |


## Contributing

Contributions are welcome! Please feel free to submit a pull request.

To run the tests for the project, use the standard `go test` command:
```sh
go test ./...
```

## License

This project is licensed under the MIT License. (Please add a LICENSE file with the actual license text).