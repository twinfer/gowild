# gowild

[![Go Reference](https://pkg.go.dev/badge/github.com/twinfer/gowild.svg)](https://pkg.go.dev/github.com/twinfer/gowild)
[![Go Report Card](https://goreportcard.com/badge/github.com/twinfer/gowild)](https://goreportcard.com/report/github.com/twinfer/gowild)

`gowild` is a lightweight optimized Go package for wildcard pattern matching. It supports multiple input types through Go generics and is designed for performance-critical applications requiring fast pattern matching on ASCII, Unicode, or binary data.

## Features

- **Fast:** Optimized algorithms 
- **Flexible Wildcards:** Supports `*`, `?`, and `.` wildcards with character classes
- **Type-Safe Generics:** Single API supporting `string` and `[]byte` types
- **Unicode Support:** Full Unicode support with proper UTF-8 character handling
- **Case-Insensitive Matching:** Built-in case-folding with Unicode support
- **Zero Allocations:** Direct `[]byte` support to minimize memory overhead
- **Efficient Complexity:** Avoids the exponential runtime of naive solutions. The time complexity is O(m*n) for pattern length m and string length n.

### Supported Wildcards

- `*`: Matches zero or more characters
- `?`: Matches zero or one character (any character)
- `.`: Matches any single character except newline
- `[abc]`: Character class matching any character in the set
- `[!abc]` or `[^abc]`: Negated character class
- `[a-z]`: Character range matching
- `\*`, `\?`, `\.`, `\[`: Escape sequences for literal characters


### Key Differences

  The main differences lie in their intended use case, wildcard behavior, and performance characteristics.

   * `path/filepath.Match`: As the name suggests, this is specifically designed for matching file paths. Its wildcards (* and ?) do not match path separators (/ or \), which is crucial for filesystem
     globbing. It's part of the standard library, so it's always available, but it's not designed for general-purpose string matching.

   * `go-wildcard`: This is a popular third-party library for general-purpose wildcard matching. It's known for being fast and feature-rich. It supports ** for matching across directory separators,
     making it a good choice for filesystem-aware matching beyond single path components.

   * `gowild` (this project): This library is designed for high-performance, general-purpose string matching. Its key differentiators are the specific behaviors of its wildcards and its focus on
     a highly optimized, unified matching algorithm.

  Feature Comparison

  Here is a table summarizing the differences:

  | Feature | gowild (this project) | path/filepath.Match | go-wildcard |
  | :--- | :--- | :--- | :--- |
  | **Primary Use Case** | High-performance general matching | File path matching | General matching |
  | **`?` Behavior** | Zero or one character | Exactly one character | Zero or one character |
  | **`.` Behavior** | Any single char except newline | Not supported | Exactly one character |
  | **`*` Behavior** | Matches any sequence (incl. /) | Matches any sequence (excl. /) | Matches zero or more chars |
  | **`**` Behavior** | Same as `*` | Not supported | Matches anything (incl. /) |
  | **Path Separators** | No special handling | Special handling (wildcards don't cross) | Special handling with `**` |
  | **Case-Insensitive** | Yes (via `MatchFold`) | OS-dependent | Yes |
  | **Performance** | High (optimized backtracking) | Moderate | High |
  | **Data Types** | `string`, `[]byte` | `string` | `string`, `[]byte` |

  Summary

   * Use `path/filepath.Match` when you need to match file paths in a way that is consistent with shell globbing, and you don't want wildcards to cross directory boundaries.
   * Use `gowild` when you need a high-performance, general-purpose library with the specific wildcard semantics it provides (? as optional, . excludes newlines), and you don't need special handling
     for path separators.


## Installation

```sh
go get github.com/twinfer/gowild
```

## Usage



```go
package main

import (
    "github.com/twinfer/gowild"
)

func main() {

    // ### Basic String Matching The `gowild.Match` function automatically optimizes based on input type:

    // String matching with ? wildcard (matches zero or one character)
    match, _ := gowild.Match("h?llo*world", "hello beautiful world")    // Output: true (? matches 'e')
    
    // ? can also match zero characters
    match, _ = gowild.Match("h?llo*world", "hllo beautiful world")  // Output: true (? matches zero characters)

    // Does not match
    match, _ = gowild.Match("h?llo*world", "goodbye world") // Output: false

    // Unicode-aware matching with strings - . matches any char except newline
    match, _ := gowild.Match("café.", "café1") // Output: true (. matches '1')

    // . does not match spaces even with Unicode
    match, _ = gowild.Match("café.", "café ")   // Output: false (. does not match space)
    
    // ? wildcard with Unicode characters (matches zero or one)
    match, _ = gowild.Match("caf?", "café") // Output: true (? matches 'é')
    
    match, _ = gowild.Match("caf?", "caf")  // Output: true (? matches zero characters)
    
    match, _ = gowild.Match("café*", "café au lait")    // Output: true (* matches ' au lait')

    // **### Byte Slice Matching** Zero-allocation matching with `[]byte`:

    pattern := []byte("*.txt")
    filename := []byte("document.txt")
    match, _ := gowild.Match(pattern, filename) // Output: true

    // ### Case-Insensitive Matching Use `MatchFold` for case-insensitive matching:
        // ASCII case-insensitive
    match, _ := gowild.MatchFold("HELLO*", "hello world")   // Output: true

    // Unicode case-insensitive with strings
    match, _ = gowild.MatchFold("CAFÉ*", "café au lait")    // Output: true
    
    // ? wildcard with case-insensitive matching (zero or one character)
    match, _ = gowild.MatchFold("caf?", "CAFÉ") // Output: true (? matches 'É')
    
    match, _ = gowild.MatchFold("caf?", "CAF")  // Output: true (? matches zero characters)

    // ### Dot Wildcard (Any Character Except Newline) The `.` wildcard is useful for matching any character while avoiding newlines:

       // . matches any character except newline
    match, _ := gowild.Match("file.txt", "file1.txt")   // Output: true (. matches '1')
    
    match, _ = gowild.Match("file.txt", "file .txt")    // Output: false (. does not match space)
    
    // Useful for identifiers and filenames
    match, _ = gowild.Match("var.", "var_") // Output: true (. matches '_')
    
    match, _ = gowild.Match("user.name", "user_name")   // Output: true (. matches '_')
    
    match, _ = gowild.Match("user.name", "user name")   // Output: false (. does not match space)

    // ### Character Classes

        // Match any vowel
    match, _ := gowild.Match("h[aeiou]llo", "hello")    // Output: true

    // Match anything except digits
    match, _ = gowild.Match("file[!0-9].txt", "fileA.txt")  // Output: true
    
    // Range matching
    match, _ = gowild.Match("[a-z][0-9]", "a5") // Output: true

}
```

## API Overview

The simplified generic API provides two main functions:

| Function       | Description                                             |
|----------------|---------------------------------------------------------|
| `Match[T]`     | Case-sensitive matching for `string` or `[]byte`        |
| `MatchFold[T]` | Case-insensitive matching for `string` or `[]byte`      |

Zero-allocation matching for binary & string data with full Unicode support


## Performance



## Contributing

Contributions are welcome! Please feel free to submit a pull request.

To run the tests for the project:
```sh
go test ./...                    # Run all tests
go test -v ./...                 # Run tests with verbose output  
go test -fuzz=FuzzMatch          # Run fuzz testing for Match function
```

## License

This project is licensed under the Apache License 2.0.