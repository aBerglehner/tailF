## Interactive replacement for -> tail -f -n 600 file.txt | grep "search"

### with the difference without options it is only highlighting the search

### No dependencies! only the go standard library

### The goal was to fast and interactive watch and search(highlight) a file without everytime when I want to change the search that I need to cancel and start the last "tail | grep" again.

```bash
-tail -n 500 file.txt | grep "search"
```

![TailF Demo](assets/demo1.gif)

## Argument options:

### main.go -n 50 file.txt -> -n specifies how many lines to watch

### Summary

```bash
| Tool                      | Read Size | Time     | Throughput  |
| ------------------------- | --------- | -------- | ----------- |
| tailF (default) 600 lines | 64 KB     | ~0.20 ms | ~300 MB/s   |
| tailF (no search)         | 3.4 MB    | ~2.5 ms  | ~1366 MB/s  |
| tailF (search)            | 3.4 MB    | ~6.0 ms  | ~560 MB/s   |
| GNU `tail | grep`         | 3.4 MB    | ~5 ms    | ~680 MB/s   |
| GNU `tail -n 600 | grep`  | 64 KB     | ~1.89 ms | ~35 MB/s    |
```

## Below I show actual results when running the program

### of course this takes longer as it prints to your terminal and that is highly dependend on the emulator

### I ran it on -> terminal: alacritty in tmux

## All benchmarks were run on:

- Linux (amd64)
- Intel® Core™ i7-8700K @ 3.70GHz
- Warm filesystem cache

# Go tools

```bash
go test -bench=. -benchmem
❯ go test -bench=. -benchmem
goos: linux
goarch: amd64
pkg: github.com/alex/tailF
cpu: Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz
BenchmarkMaxLines-12                         216           5698711 ns/op                 3.418 inputSize/mb              5.699 ms/op           599.8 throughput/MBps    15174319 B/op         51 allocs/op
BenchmarkQuaterOfLines-12                    656           1860776 ns/op                 0.8545 inputSize/mb             1.861 ms/op           459.2 throughput/MBps     3803703 B/op         51 allocs/op
BenchmarkDefaultLines-12                    5306            215290 ns/op                 0.06409 inputSize/mb            0.2153 ms/op          297.7 throughput/MBps      295595 B/op         18 allocs/op
BenchmarkMaxLinesNoSearch-12                 482           2464460 ns/op                 3.418 inputSize/mb              2.464 ms/op          1387 throughput/MBps       7178754 B/op         39 allocs/op
PASS
ok      github.com/alex/tailF   9.561s


```

### For the tests you need to run createTestFile.sh first -> this will create big.txt

# GNU tools

```bash
#actually running tailF and searching on max size
# terminal: alacritty in tmux
❯ sudo perf stat ./tailF -n 32000 big.txt

Performance counter stats for './tailF -n 32000 big.txt':

            522,21 msec task-clock                       #    0,028 CPUs utilized
             2.159      context-switches                 #    4,134 K/sec
               130      cpu-migrations                   #  248,942 /sec
            23.035      page-faults                      #   44,111 K/sec
     2.213.718.197      cycles                           #    4,239 GHz
     2.168.253.932      instructions                     #    0,98  insn per cycle
       598.885.613      branches                         #    1,147 G/sec
         4.338.719      branch-misses                    #    0,72% of all branches

      18,599828021 seconds time elapsed

       0,110033000 seconds user
       0,415256000 seconds sys


# actually running tailF and searching on default lines(600)
# terminal: alacritty in tmux
❯ sudo perf stat ./tailF big.txt

 Performance counter stats for './tailF big.txt':

             13,71 msec task-clock                       #    0,001 CPUs utilized
               296      context-switches                 #   21,589 K/sec
                13      cpu-migrations                   #  948,158 /sec
             1.029      page-faults                      #   75,050 K/sec
        54.331.926      cycles                           #    3,963 GHz
        49.119.135      instructions                     #    0,90  insn per cycle
        12.688.234      branches                         #  925,419 M/sec
           138.063      branch-misses                    #    1,09% of all branches

      11,989157459 seconds time elapsed

       0,001960000 seconds user
       0,012745000 seconds sys





# more extended stat version from summary above
❯ sudo perf stat -r 50 \
  sh -c 'tail -n 32000 big.txt | grep "th" > /dev/null'


 Performance counter stats for 'sh -c tail -n 32000 big.txt | grep "th" > /dev/null' (50 runs):

              5,21 msec task-clock                       #    1,367 CPUs utilized               ( +-  0,34% )
               159      context-switches                 #   30,544 K/sec                       ( +-  0,63% )
                 0      cpu-migrations                   #    0,000 /sec
               299      page-faults                      #   57,439 K/sec                       ( +-  0,32% )
        20.865.255      cycles                           #    4,008 GHz                         ( +-  0,33% )
        15.588.783      instructions                     #    0,75  insn per cycle              ( +-  0,11% )
         2.905.234      branches                         #  558,103 M/sec                       ( +-  0,10% )
            56.881      branch-misses                    #    1,96% of all branches             ( +-  3,00% )

         0,0038091 +- 0,0000162 seconds time elapsed  ( +-  0,43% )



# more extended stat version from summary above
❯ sudo perf stat -r 50 \
  sh -c 'tail -n 600 big.txt | grep "th" > /dev/null'


 Performance counter stats for 'sh -c tail -n 600 big.txt | grep "th" > /dev/null' (50 runs):

              1,89 msec task-clock                       #    1,192 CPUs utilized               ( +-  0,39% )
                 3      context-switches                 #    1,585 K/sec                       ( +-  1,35% )
                 0      cpu-migrations                   #    0,000 /sec
               303      page-faults                      #  160,134 K/sec                       ( +-  0,12% )
         8.033.646      cycles                           #    4,246 GHz                         ( +-  0,40% )
         6.802.401      instructions                     #    0,85  insn per cycle              ( +-  0,06% )
         1.240.786      branches                         #  655,751 M/sec                       ( +-  0,06% )
            42.140      branch-misses                    #    3,40% of all branches             ( +-  1,25% )

        0,00158698 +- 0,00000636 seconds time elapsed  ( +-  0,40% )

```
