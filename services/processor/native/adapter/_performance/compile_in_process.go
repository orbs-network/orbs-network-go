// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package main

import (
	gcamd64 "cmd/compile/internal/amd64"
	"cmd/compile/internal/gc"
	ldamd64 "cmd/link/internal/amd64"
	"cmd/link/internal/ld"
	"flag"
	"os"
	"strings"
)

func main() {

	// # import config (importcfg)
	// packagefile github.com/orbs-network/orbs-contract-sdk/go/sdk=$WORK/b002/_pkg_.a
	// packagefile runtime=/usr/local/go/pkg/darwin_amd64_dynlink/runtime.a
	// packagefile runtime/cgo=/usr/local/go/pkg/darwin_amd64_dynlink/runtime/cgo.a

	fixCompileFlags()
	compile()
	fixLinkFlags()
	link()
}

func compile() {
	// compile -o $WORK/b001/_pkg_.a -trimpath $WORK/b001 -dynlink -p plugin/unnamed-863cfd5c3f9230dc004118e0c85719727de9eeeb -complete -installsuffix dynlink -buildid wN1Zdxgt7mzbFxCeLK1s/wN1Zdxgt7mzbFxCeLK1s -goversion go1.10.2 -D _/Users/talkol/go/src/github.com/orbs-network/orbs-network-go/services/processor/native/adapter/performance -importcfg $WORK/b001/importcfg -pack ./counter100.go
	os.Args = strings.Split("compile -o counter100.a -dynlink -p ./plugin -complete -installsuffix dynlink -importcfg ./import.cfg -pack ./counter100.go", " ")
	gc.Main(gcamd64.Init)
}

func link() {
	// link -o $WORK/b001/exe/a.out.so -importcfg $WORK/b001/importcfg.link -installsuffix dynlink -pluginpath plugin/unnamed-02e400643c4db48c87933ffd5ec9a025a079f5da -buildmode=plugin -buildid=fkf3T4KmTpXXnsnunUs2/wN1Zdxgt7mzbFxCeLK1s/N6YKIPnGYXp4WfH4sQly/fkf3T4KmTpXXnsnunUs2 -w -extld=clang $WORK/b001/_pkg_.a
	os.Args = strings.Split("link -o counter100.so -importcfg ./import.cfg -installsuffix dynlink -pluginpath ./plugin -buildmode=plugin -w -extld=clang ./counter100.a", " ")
	arch, theArch := ldamd64.Init()
	ld.Main(arch, theArch)
}

func fixCompileFlags() {
	flag.CommandLine = flag.NewFlagSet("compile", flag.ExitOnError)
}

func fixLinkFlags() {
	// required because cmd/link/internal/ld/main.go defines them globally as static vars
	flag.CommandLine = flag.NewFlagSet("link", flag.ExitOnError)
	flag.String("buildid", "", "record `id` as Go toolchain build id")

	flag.String("o", "", "write output to `file`")
	flag.String("pluginpath", "", "full path name for plugin")

	flag.String("installsuffix", "", "set package directory `suffix`")
	flag.Bool("dumpdep", false, "dump symbol dependency graph")
	flag.Bool("race", false, "enable race detector")
	flag.Bool("msan", false, "enable MSan interface")

	flag.String("k", "", "set field tracking `symbol`")
	flag.String("libgcc", "", "compiler support lib for internal linking; use \"none\" to disable")
	flag.String("tmpdir", "", "use `directory` for temporary files")

	flag.String("extld", "", "use `linker` when linking in external mode")
	flag.String("extldflags", "", "pass `flags` to external linker")
	flag.String("extar", "", "archive program for buildmode=c-archive")

	flag.Bool("a", false, "disassemble output")
	flag.Bool("c", false, "dump call graph")
	flag.Bool("d", false, "disable dynamic executable")
	flag.Bool("f", false, "ignore version mismatch")
	flag.Bool("g", false, "disable go package data checks")
	flag.Bool("h", false, "halt on error")
	flag.Bool("n", false, "dump symbol table")
	flag.Bool("s", false, "disable symbol table")
	flag.Bool("u", false, "reject unsafe packages")
	flag.Bool("w", false, "disable DWARF generation")

	flag.String("I", "", "use `linker` as ELF dynamic linker")
	flag.Int("debugtramp", 0, "debug trampolines")

	flag.Int("R", -1, "set address rounding `quantum`")
	flag.Int64("T", -1, "set text segment `address`")
	flag.Int64("D", -1, "set data segment `address`")
	flag.String("E", "", "set `entry` symbol name")

	flag.String("cpuprofile", "", "write cpu profile to `file`")
	flag.String("memprofile", "", "write memory profile to `file`")
	flag.Int64("memprofilerate", 0, "set runtime.MemProfileRate to `rate`")
}
