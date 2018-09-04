# SputnikVM FFI Bindings

This repo contains C and Go bindings for the SputnikVM core
library.

## Usage

### C

In `c` folder, run `make build`. It will generate an object file
`libsputnikvm.so`, and you can use the header file `sputnikvm.h` to
interact with it. You can find the generated documentation file for
`sputnikvm.h`
[here](https://ethereumproject.github.io/sputnikvm-ffi/sputnikvm_8h.html).

### Go

<img src="./go/gopher.png" width="100" height="100" />

Import the `sputnikvm` library to your application:

```
import "github.com/ethereumproject/sputnikvm-ffi/go/sputnikvm"
```

Build a static library for the C FFI, which will give you an
`libsputnikvm.a` file:

```
cd c
make build
```

When building your Go application, pass `CGO_LDFLAGS` to link the C
library.

```
CGO_LDFLAGS="/path/to/libsputnikvm.a -ldl" go build .
```

Refer to
[GoDoc](https://godoc.org/github.com/ethereumproject/sputnikvm-ffi/go/sputnikvm)
for documentation of the Go bindings.
