#!/bin/sh

rm -rf /tmp/*.prof

go test ./test/_manual -count 1

go tool pprof --inuse_space -nodecount 10 -weblist orbs-network-go -hide /orbs-network-go/test/ --base /tmp/mem-tx-before.prof /tmp/mem-tx-after.prof
go tool pprof --inuse_space -nodecount 20 -weblist orbs-network-go -hide /orbs-network-go/test/ --base /tmp/mem-shutdown-before.prof /tmp/mem-shutdown-after.prof

echo ""
echo ""
echo "TestMemoryLeaks_AfterSomeTransactions:"
echo ""

go tool pprof --inuse_space -nodecount 10 -top -show orbs-network-go -hide /orbs-network-go/test/ --base /tmp/mem-tx-before.prof /tmp/mem-tx-after.prof

echo ""
echo ""
echo "TestMemoryLeaks_OnSystemShutdown:"
echo ""

go tool pprof --inuse_space -nodecount 20 -top  --base /tmp/mem-shutdown-before.prof /tmp/mem-shutdown-after.prof