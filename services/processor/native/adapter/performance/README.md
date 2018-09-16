# Native Go Contract Compiler Performance

## Why is this important

When possible, we would like native contract compilation to be serial to the consensus process. Since consensus over native Go contracts is achieved over their source code, the preferable approach is lazy compilation of the source code as needed, right before execution. This will allow every node to maintain its own cache of compiled artifacts. New nodes will only compile contracts as needed when they're required to run them.

The difficulty with the lazy compilation approach is that the first compilation takes place during the consensus process and may cause the compiling node to miss its consensus time slot. Therefore, we would like compilation to take place as quickly as possible.

## Optimization research

We would like to investigate multiple potential optimizations aimed at reducing compilation time. Note that debugging the build process is possible by adding the `-x` flag.  

#### Baseline

Go 1.10 caches build artifacts by [default](https://tip.golang.org/doc/go1.10#build). Experiments show that the first build of any plugin takes about 1600ms but subsequent builds of other plugins take about 220ms.

Test: `./baseline.sh`

#### Reducing compiler optimizations

Effect is very small. Reduces first build by about 100ms and subsequent builds by about 10ms.

Test: `./no_optimizations`

#### Adding memory to compiler toolchain

Effect is somewhat small. Reduces first build by about 100ms and subsequent builds by about 20ms.

See: https://www.reddit.com/r/golang/comments/476pae/how_to_speed_up_go_compiler_and_many_other_go/

Test: `./more_memory.sh`

#### Caching intermediate package builds

Many stdlib packages need to be rebuilt by the toolchain configured for dynamic linking (like `runtime`), and by default `go build` does not cache intermediate package builds.

It seems that Go 1.10 does this by default using `cache` but adding `-i` does this even more aggressively in a way that's difficult to undo.

See: https://github.com/golang/go/issues/19707

See: https://tip.golang.org/doc/go1.10#build

Test: `./cache_itermediate.sh`

#### Running the go compiler in-process

There is overhead in starting a new process for each `go build`. We could attempt to avoid it by running the entire build toolchain in-process with the processor itself.

See: https://github.com/golang/go/blob/master/src/cmd/compile/main.go

See: https://github.com/golang/go/blob/master/src/cmd/link/main.go

#### Trying alternative compilers

Go version 1.5 converted the compiler to Go from C, possibly making builds slower. Was unable to test `gccgo` compiler on OSX due to problems installing `gccgo`. Need to check on Linux.

See: https://golang.org/doc/go1.5#c

Test: `./other_compiler.sh`

#### Dynamically linking standard libraries

The resulting shared object is 800KB in size due to static linking to the standard libraries. Was unable to test this on OSX due to the `-linkshared` linker flag only working on ELF systems (Linux).

See: https://golang.org/s/execmodes

Test: `./dynamic_link.sh`

#### Experimenting with various build flags

See: https://golang.org/cmd/compile/

See: https://golang.org/cmd/link/
