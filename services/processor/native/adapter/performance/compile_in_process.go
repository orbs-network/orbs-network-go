package main

import (
	"cmd/compile/internal/gc"
	gcamd64 "cmd/compile/internal/amd64"
	"cmd/link/internal/ld"
	ldamd64 "cmd/link/internal/amd64"
	"os"
	"strings"
	)

func main() {

	// # import config (importcfg)
	// packagefile github.com/orbs-network/orbs-contract-sdk/go/sdk=$WORK/b002/_pkg_.a
	// packagefile runtime=/usr/local/go/pkg/darwin_amd64_dynlink/runtime.a
	// packagefile runtime/cgo=/usr/local/go/pkg/darwin_amd64_dynlink/runtime/cgo.a

	compile()
	link()
}

func compile() {
	// compile -o $WORK/b001/_pkg_.a -trimpath $WORK/b001 -dynlink -p plugin/unnamed-863cfd5c3f9230dc004118e0c85719727de9eeeb -complete -installsuffix dynlink -buildid wN1Zdxgt7mzbFxCeLK1s/wN1Zdxgt7mzbFxCeLK1s -goversion go1.10.2 -D _/Users/talkol/go/src/github.com/orbs-network/orbs-network-go/services/processor/native/adapter/performance -importcfg $WORK/b001/importcfg -pack ./counter100.go
	os.Args = strings.Split("compile -dynlink -complete -importcfg ./import.cfg ./counter100.go", " ")
	gc.Main(gcamd64.Init)
}

func link() {
	// link -o $WORK/b001/exe/a.out.so -importcfg $WORK/b001/importcfg.link -installsuffix dynlink -pluginpath plugin/unnamed-02e400643c4db48c87933ffd5ec9a025a079f5da -buildmode=plugin -buildid=fkf3T4KmTpXXnsnunUs2/wN1Zdxgt7mzbFxCeLK1s/N6YKIPnGYXp4WfH4sQly/fkf3T4KmTpXXnsnunUs2 -w -extld=clang $WORK/b001/_pkg_.a
	os.Args = strings.Split("link -o counter100.so -importcfg ./import.cfg -buildmode=plugin -w -extld=clang ./counter100.o", " ")
	arch, theArch2 := ldamd64.Init()
	ld.Main(arch, theArch2)
}