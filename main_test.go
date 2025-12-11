package main

import (
	"bufio"
	"fmt"
	"os"
	"testing"
)

func BenchmarkNormal(b *testing.B) {
	fileName := "/var/log/syslog"
	// fileName := "big.txt"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		run(fileName, MaxReadLineCount)
	}

	b.StopTimer()

	nsPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	secondsPerOp := float64(nsPerOp) * 1e-9

	byteSize, err := lastNLinesSize(fileName, int(MaxReadLineCount))
	if err != nil {
		return
	}
	mbSize := float64(byteSize) / 1024 / 1024
	fmt.Printf("mbSize: %v mb\n", mbSize)

	throughputBytesPerSec := float64(byteSize) / secondsPerOp
	throughputMBps := throughputBytesPerSec / (1024 * 1024)

	b.ReportMetric(nsPerOp/1e6, "ms/op")
	b.ReportMetric(throughputMBps, "throughput/MBps")
}

func lastNLinesSize(path string, n int) (int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:]
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}

	var totalSize int64
	for _, line := range lines {
		totalSize += int64(len(line)) + 1 // +1 for '\n'
	}

	return totalSize, nil
}
