sqrt
====

A package to compute square roots and cube roots to arbitrary precision.

This package is dedicated to my mother, who taught me how to calculate square roots by hand as a child.

## How this package differs from big.Float.Sqrt

big.Float.Sqrt requires a finite precision to be set ahead of time. The answer it gives is only accurate to that precision. This package does not require a precision to be set in advance. Square root values in this package compute their digits lazily on an as needed basis just as one would compute square roots by hand. Also, this package features cube roots which big.Float in the standard library does not offer as of this writing. Cube root values in this package also compute their digits lazily on an as needed basis just as one would compute cube roots by hand.

## Examples

```golang
package main

import (
    "fmt"

    "github.com/keep94/sqrt"
)

func main() {
    var ctxt sqrt.Context
    defer ctxt.Close()

    // Print the first 1000 digits of the square root of 2.
    fmt.Printf("%.1000g\n", ctxt.Sqrt(2))

    // Print the 10,000th digit of the cube root of 5.
    fmt.Println(ctxt.CubeRoot(5).At(9999))
}
```

## Related Repos

- [github.com/keep94/numprint](https://github.com/keep94/numprint)
- [github.com/keep94/numsearch](https://github.com/keep94/numsearch)

## More Documentation

More documentation and examples can be found [here](https://pkg.go.dev/github.com/keep94/sqrt).
