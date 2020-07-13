package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/blockstorage/adapter/filesystem"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"os"
)

const DefaultMaxBlockSize = 64 * 1024 * 1024
const MB = 1024 * 1024

var fs = flag.NewFlagSet("", flag.ContinueOnError)

func usage() {
	fmt.Fprintf(fs.Output(), "\nOrbs Blocks file analysis tool\n\nUsage: %s [options] blocks_file_name\n\nOptions:\n", os.Args[0])
	fs.PrintDefaults()
}

func init() {
	fs.Usage = usage
}

func main() {
	maxBlockSizeBytes := fs.Uint64("max-block-size", DefaultMaxBlockSize, "maximum block size in bytes (64MB)")
	help := fs.Bool("help", false, "print options")
	fs.Parse(os.Args[1:])
	filepath := fs.Arg(0)

	if *help {
		fs.Usage()
		os.Exit(0)
	}

	if filepath == "" {
		fs.Usage()
		os.Exit(1)
	}
	analyze(filepath, *maxBlockSizeBytes)
}

func analyze(filepath string, maxBlockSizeBytes uint64) {
	dp, err := filesystem.NewDiagnosticsParser(filepath)
	if err != nil {
		fs.Usage()
		fmt.Printf("\n\nError opening file %s for diagnostics: %e", filepath, err)
		os.Exit(1)
	}
	defer dp.Close()

	fmt.Printf("\n\nBlocks file:  %s\nFile Version: %d\nNetwork Type: %d\nChainId:      %d\n\n", filepath, dp.Header.FileVersion, dp.Header.NetworkType, dp.Header.ChainId)

	scanAllBlocks(maxBlockSizeBytes, dp, filepath)
}

func scanAllBlocks(maxBlockSizeBytes uint64, dp *filesystem.DiagnosticsParser, filepath string) {
	registry := metric.NewRegistry()
	h := registry.NewHistogramInt64("blocksSize", int64(maxBlockSizeBytes))

	totalSize := dp.FileInfo.Size()

	pb := newProgressBar("Scanning Blocks", totalSize, 30)

	err := dp.ScanFile(uint32(maxBlockSizeBytes), func(size int, offset int64, block *protocol.BlockPairContainer) {
		h.Record(int64(size))

		// TODO collect more interesting stats

		pb.updateProgressBar(offset)
	})

	if err != nil {
		fs.Usage()
		fmt.Printf("\n\nError scanning file %s: %e", filepath, err)
		os.Exit(1)
	}

	pb.doneProgressBar()

	bytes, _ := json.MarshalIndent(registry.ExportAll(), "", "  ")
	fmt.Print("Stats:\n\n", string(bytes))
}

// Simple progress bar

type progressBar struct {
	width       int
	targetValue int64
	stepValue   int
	progress    int
}

func newProgressBar(label string, _totalSize int64, width int) *progressBar {

	fmt.Printf("%s [\033[%dC]\033[%dD", label, width, width+1) // draw the empty progress bar.
	return &progressBar{
		targetValue: _totalSize,
		stepValue:   int(_totalSize) / width,
		progress:    0,
	}
}

func (pb *progressBar) doneProgressBar() {
	pb.updateProgressBar(pb.targetValue)
	fmt.Printf("] Done!\n") // complete the progress bar
}

func (pb *progressBar) updateProgressBar(currentValue int64) {
	delta := int(currentValue)/pb.stepValue - pb.progress
	for i := 0; i < delta; i++ {
		fmt.Print(".")
		pb.progress++
	}

}
