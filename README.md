## Interactive replacement for -> tail -f -n file.txt | grep "search"

### No dependencies! only the go standard library

### The goal was to enable fast, interactive file watching and searching without restarting tail | grep for every search change.

```bash
-tail -n file.txt | grep "search"
```

### the search has 2 delimiters when it stops:

-n how many searches(lines) it will print -> default 500
-t specifies the approximate wait time, based on how fast the file can be read.
Default: 50 ms, which corresponds to ~60 MB on my machine.

### and of course if it read the whole file

### One current limitation

If new lines keep arriving, you have a 5-second window to type and submit your search query. In most cases this is more than sufficient.
(otherwise learn to type)

![TailF Demo](assets/demo_0.2.0.gif)

## Argument options:

### -t max search time in ms only relevent in normal (grep) mode

### -n number of lines to search

### -h print all lines and highlight search

### main.go -n 50 file.txt

### main.go -g file.txt

### Summary

```bash
| Tool                          | Read Size | Time      | Throughput  |
| ----------------------------- | --------- | --------- | ----------- |
| tailF (big file)              | 60 MB     | ~50.29 ms | ~1192 MB/s  |
| tailF (small file)            | 50 KB     | ~0.15 ms  | ~353 MB/s   |
| GNU `tail -n 500_000 | grep`  | 60 MB     | ~64.91 ms | ~924 MB/s   |
```

## Below I show actual results when running the program

### of course this takes longer as it prints to your terminal and that is highly dependend on the emulator

### I ran it on -> terminal: alacritty in tmux

### go build :

❯ CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -trimpath -o tailF_0.2.0 main.go

## All benchmarks were run on:

- Linux (amd64)
- Intel® Core™ i7-8700K @ 3.70GHz
- Warm filesystem cache

# Go tools

```bash
❯ go test -bench=. -benchmem
goos: linux
goarch: amd64
pkg: github.com/alex/tailF
cpu: Intel(R) Core(TM) i7-8700K CPU @ 3.70GHz
BenchmarkDefault_GrepFlag_Newscrawl-12                24          50328332 ns/op                60.00 inputSize/mb              50.33 ms/op           1192 throughput/MBps      62969423 B/op         58 allocs/op
BenchmarkDefaultLines_GrepFlag-12                   8144            151174 ns/op                 0.05341 inputSize/mb            0.1512 ms/op          353.3 throughput/MBps      180761 B/op         15 allocs/op
BenchmarkMaxLinesNoSearch_GrepFlag-12                541           2236468 ns/op                 3.418 inputSize/mb              2.236 ms/op          1528 throughput/MBps       3588473 B/op          8 allocs/op
BenchmarkMaxLines_Highlight-12                       195           6194443 ns/op                 3.418 inputSize/mb              6.194 ms/op           551.8 throughput/MBps    15174278 B/op         51 allocs/op
BenchmarkMaxLinesNoSearch_Highlight-12               504           2320043 ns/op                 3.418 inputSize/mb              2.320 ms/op          1473 throughput/MBps       3588472 B/op          8 allocs/op
BenchmarkDefaultLines_Highlight-12                  5430            220324 ns/op                 0.05341 inputSize/mb            0.2203 ms/op          242.4 throughput/MBps      246297 B/op         14 allocs/op


```

### For the tests you need to run createTestFile.sh first -> this will create ./assets/testFiles/big.txt

### the eng_newscrawl_2018_1M file you can find here -> https://wortschatz.uni-leipzig.de/en/download/eng -> test run search was "suspicious"

# GNU tools

```bash

# actually running tailF and searching on default lines(500)
# terminal: alacritty in tmux
❯ sudo perf stat ./tailF_0.2.0  assets/testFiles/eng_newscrawl_2018_1M-sentences.txt
 Performance counter stats for './tailF_0.2.0 assets/testFiles/eng_newscrawl_2018_1M-sentences.txt':

            256,17 msec task-clock                       #    0,017 CPUs utilized
               829      context-switches                 #    3,236 K/sec
                64      cpu-migrations                   #  249,836 /sec
             3.696      page-faults                      #   14,428 K/sec
     1.089.845.698      cycles                           #    4,254 GHz
     1.901.182.160      instructions                     #    1,74  insn per cycle
       461.089.517      branches                         #    1,800 G/sec
         7.260.023      branch-misses                    #    1,57% of all branches

      15,152222258 seconds time elapsed

       0,224853000 seconds user
       0,032952000 seconds sys




# more extended stat version from summary above
❯ sudo perf stat -r 50 \
  sh -c 'tail -n 500000 ./assets/testFiles/eng_newscrawl_2018_1M-sentences.txt | grep "suspicious" > /dev/null'

 Performance counter stats for 'sh -c tail -n 500000 ./assets/testFiles/eng_newscrawl_2018_1M-sentences.txt | grep "suspicious" > /dev/null' (50 runs):

             64,91 msec task-clock                       #    1,380 CPUs utilized               ( +-  0,13% )
             2.939      context-switches                 #   45,276 K/sec                       ( +-  0,17% )
                 0      cpu-migrations                   #    0,000 /sec
               311      page-faults                      #    4,791 K/sec                       ( +-  0,11% )
       252.410.081      cycles                           #    3,888 GHz                         ( +-  0,14% )
       159.038.512      instructions                     #    0,63  insn per cycle              ( +-  0,10% )
        29.765.307      branches                         #  458,543 M/sec                       ( +-  0,08% )
           854.336      branch-misses                    #    2,87% of all branches             ( +-  0,45% )

         0,0470445 +- 0,0000546 seconds time elapsed  ( +-  0,12% )


```
