# gowild

[![Go Reference](https://pkg.go.dev/badge/github.com/twinfer/gowild.svg)](https://pkg.go.dev/github.com/twinfer/gowild)
[![Go Report Card](https://goreportcard.com/badge/github.com/twinfer/gowild)](https://goreportcard.com/report/github.com/twinfer/gowild)

`gowild` is a lightweight and highly optimized Go package for wildcard pattern matching. It supports multiple input types through Go generics and is designed for performance-critical applications requiring fast pattern matching on ASCII, Unicode, or binary data.

## Features

- **Fast:** Optimized algorithms with consecutive wildcard grouping to avoid exponential complexity
- **Flexible Wildcards:** Supports `*`, `?`, and `.` wildcards with character classes
- **Type-Safe Generics:** Single API supporting `string`, `[]byte`, and `[]rune` types
- **Unicode Support:** Automatic Unicode-aware matching for `[]rune` inputs
- **Case-Insensitive Matching:** Built-in case-folding with Unicode support
- **Zero Allocations:** Direct `[]byte` support to minimize memory overhead
- **Efficient Complexity:** Avoids the exponential runtime of naive solutions. The worst-case time complexity is O(m*n) for pattern length m and string length n.

### Supported Wildcards

- `*` - Matches any sequence of characters (including zero characters)
- `?` - Matches any single character or zero characters (optional character)  
- `.` - Matches any single character (required character)
- `[abc]` - Matches any character in the set (a, b, or c)
- `[!abc]` or `[^abc]` - Matches any character not in the set
- `[a-z]` - Matches any character in the range a to z
- `\*`, `\?`, `\.`, `\[` - Matches the literal character

## Installation

```sh
go get github.com/twinfer/gowild
```

## Usage

### Basic String Matching

The `gowild.Match` function automatically optimizes based on input type:

```go
package main

import (
    "fmt"
    "github.com/twinfer/gowild"
)

func main() {
    // String matching (optimized for ASCII)
    match, _ := gowild.Match("h?llo*world", "hello beautiful world")
    fmt.Println(match) // Output: true

    // Does not match
    match, _ = gowild.Match("h?llo*world", "goodbye world") 
    fmt.Println(match) // Output: false
}
```

### Unicode Matching

For Unicode strings, use `[]rune` type for correct multi-byte character handling:

```go
package main

import (
    "fmt"
    "github.com/twinfer/gowild"
)

func main() {
    // Unicode-aware matching with []rune
    pattern := []rune("caf.")
    input := []rune("café")
    match, _ := gowild.Match(pattern, input)
    fmt.Println(match) // Output: true

    // String matching treats multi-byte chars as separate bytes
    match, _ = gowild.Match("caf.", "café")
    fmt.Println(match) // Output: false
}
```

### Byte Slice Matching

Zero-allocation matching with `[]byte`:

```go
package main

import (
    "fmt"
    "github.com/twinfer/gowild"
)

func main() {
    pattern := []byte("*.txt")
    filename := []byte("document.txt")
    match, _ := gowild.Match(pattern, filename)
    fmt.Println(match) // Output: true
}
```

### Case-Insensitive Matching

Use `MatchFold` for case-insensitive matching:

```go
package main

import (
    "fmt"
    "github.com/twinfer/gowild"
)

func main() {
    // ASCII case-insensitive
    match, _ := gowild.MatchFold("HELLO*", "hello world")
    fmt.Println(match) // Output: true

    // Unicode case-insensitive
    pattern := []rune("CAFÉ*")
    input := []rune("café au lait") 
    match, _ = gowild.MatchFold(pattern, input)
    fmt.Println(match) // Output: true
}
```

### Character Classes

```go
package main

import (
    "fmt"
    "github.com/twinfer/gowild"
)

func main() {
    // Match any vowel
    match, _ := gowild.Match("h[aeiou]llo", "hello")
    fmt.Println(match) // Output: true

    // Match anything except digits
    match, _ = gowild.Match("file[!0-9].txt", "fileA.txt")
    fmt.Println(match) // Output: true
    
    // Range matching
    match, _ = gowild.Match("[a-z][0-9]", "a5")
    fmt.Println(match) // Output: true
}
```

## API Overview

The simplified generic API provides two main functions:

| Function    | Description                                                           |
|-------------|-----------------------------------------------------------------------|
| `Match[T]`  | Case-sensitive matching for `string`, `[]byte`, or `[]rune`         |
| `MatchFold[T]` | Case-insensitive matching for `string`, `[]byte`, or `[]rune`    |

### Type-Specific Optimizations

- **`string`**: Fast byte-wise matching optimized for ASCII
- **`[]byte`**: Zero-allocation matching for binary data
- **`[]rune`**: Unicode-aware matching for multi-byte characters

The functions automatically select the optimal matching strategy based on the input type.

## Performance

The package includes several performance optimizations:

- **Fast paths** for simple patterns (`*`, `prefix*`, `*suffix`, `prefix*suffix`)
- **Exact matching** bypass for patterns without wildcards  
- **Consecutive wildcard optimization** to prevent exponential complexity
- **Zero-allocation** case-folding for ASCII strings
- **Iterative algorithm** with optimized backtracking

## Contributing

Contributions are welcome! Please feel free to submit a pull request.

To run the tests for the project:
```sh
go test ./...                    # Run all tests
go test -v ./...                 # Run tests with verbose output  
go test -fuzz=FuzzMatch          # Run fuzz testing
```

## License

This project is licensed under the MIT License.