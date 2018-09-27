# Manual Tests

Manual tests are tests that shouldn't run automatically in the CI. They are usually specific tests which require manual human investigation.

It is possible that as we gather more evidence, some of these tests will become automated.

### Memory leaks

This test attempts to show the top memory consumers of a running acceptance node. The test shows a report of all memory consumers and the amount of memory they currently have in-use.

* Run with `PROJECT-ROOT/test.manual-memory-leaks.sh`

* Test scenarios:

  * `TestMemoryLeaks_AfterSomeTransactions`
  
    Starts an acceptance node, GC, heap snapshot of in-use memory, send 500 transactions, GC, another heap snapshot of in-use memory. The report shows the delta between the two snapshots.  
  
  * `TestMemoryLeaks_OnSystemShutdown` 
  
    Heap snapshot of in-use memory before node is started, starts an acceptance node, sends 5 transactions, graceful shutdown of node, GC, another heap snapshot of in-use memory. The report shows the delta between the two snapshots.

* Generated reports:
  * HTML based report of places in the code base and how much memory they have allocated (delta between snapshots)
  * Console based report of top memory consumers (delta between snapshots)
  * Notes about the reports:  
    * Show delta of **in-use** memory (only what's not freed)
    * `flat` is how much memory this specific line has allocated
    * `cum` is how much memory this specific line plus methods it called have allocated