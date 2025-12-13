package main

import (
	"bufio"
	"os"
	"testing"
)

const FileName = "big.txt"

// const FileName = "/var/log/syslog"

func BenchmarkFullSize(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		run(FileName, MaxReadLineCount, "th")
	}

	b.StopTimer()

	nsPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	secondsPerOp := float64(nsPerOp) * 1e-9

	byteSize, err := lastNLinesSize(FileName, int(MaxReadLineCount))
	if err != nil {
		return
	}

	throughputBytesPerSec := float64(byteSize) / secondsPerOp
	throughputMBps := throughputBytesPerSec / (1024 * 1024)
	mbSize := float64(byteSize) / 1024 / 1024

	b.ReportMetric(nsPerOp/1e6, "ms/op")
	b.ReportMetric(throughputMBps, "throughput/MBps")
	b.ReportMetric(mbSize, "inputSize/mb")
}

func BenchmarkQuaterOfSize(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		run(FileName, MaxReadLineCount/4, "th")
	}

	b.StopTimer()

	nsPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	secondsPerOp := float64(nsPerOp) * 1e-9

	byteSize, err := lastNLinesSize(FileName, int(MaxReadLineCount)/4)
	if err != nil {
		return
	}

	throughputBytesPerSec := float64(byteSize) / secondsPerOp
	throughputMBps := throughputBytesPerSec / (1024 * 1024)
	mbSize := float64(byteSize) / 1024 / 1024

	b.ReportMetric(nsPerOp/1e6, "ms/op")
	b.ReportMetric(throughputMBps, "throughput/MBps")
	b.ReportMetric(mbSize, "inputSize/mb")
}

func Benchmark1000LinesSyslog(b *testing.B) {
	fileName := "/var/log/syslog"
	var readLineCount int16 = 1000
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		run(fileName, readLineCount, "th")
	}

	b.StopTimer()

	nsPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	secondsPerOp := float64(nsPerOp) * 1e-9

	byteSize, err := lastNLinesSize(fileName, int(readLineCount))
	if err != nil {
		return
	}

	throughputBytesPerSec := float64(byteSize) / secondsPerOp
	throughputMBps := throughputBytesPerSec / (1024 * 1024)
	mbSize := float64(byteSize) / 1024 / 1024

	b.ReportMetric(nsPerOp/1e6, "ms/op")
	b.ReportMetric(throughputMBps, "throughput/MBps")
	b.ReportMetric(mbSize, "inputSize/mb")
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
