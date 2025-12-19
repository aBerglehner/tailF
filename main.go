package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	MaxReadLineCount                    int16 = 32000
	DefaultLineCount                    int   = 500
	PredictedGeneralThroughputPerSecond int   = 1200
	WaitingTimeMs                       int   = 50
	BytesThroughputPerMs                int   = PredictedGeneralThroughputPerSecond * 1024 * 1024 / 1000
	TotalThroughput                     int   = (BytesThroughputPerMs * WaitingTimeMs)
)

var (
	highlightColor = []byte("\033[31m")
	resetColor     = []byte("\033[0m")
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("file argument missing! sample: main.go test.txt")
		return
	}
	fmt.Printf("os.Args[1]: %v\n", os.Args[1])
	if len(os.Args) == 1 && os.Args[1] == "options" {
		fmt.Println("see options")
		return
	}
	fileName := os.Args[len(os.Args)-1]
	n := flag.Int("n", DefaultLineCount, "Number of lines")
	grepOnly := flag.Bool("g", false, "print only lines that match")
	flag.Parse()

	fmt.Printf("grepOnly: %v\n", *grepOnly)

	readLineCount := int16(min(*n, int(MaxReadLineCount)))

	initSearchCh := make(chan string)
	go func() {
		reader := bufio.NewReader(os.Stdin)
		// to init read the file
		initSearchCh <- ""
		for {
			text, _ := reader.ReadString('\n')
			text = strings.TrimSpace(text)
			initSearchCh <- text
		}
	}()

	// PERF: might be able to give os.Open also to SearchLastNLines
	f, err := os.Open(fileName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	tailCh := make(chan string)
	go tailF(f, fileName, tailCh)

	var currentSearch string
	var mu sync.Mutex

	for {
		select {
		case s := <-tailCh:
			mu.Lock()
			search := currentSearch
			mu.Unlock()

			if *grepOnly && strings.Contains(s, search) {
				fmt.Printf("%s",
					strings.ReplaceAll(s, search, string(highlightColor)+search+string(resetColor)))
			}
			if !*grepOnly {
				fmt.Printf("%s",
					strings.ReplaceAll(s, search, string(highlightColor)+search+string(resetColor)))
			}
		case searchTerm := <-initSearchCh:
			mu.Lock()
			currentSearch = searchTerm
			mu.Unlock()
			// \033[H  -> moves the cursor to top-left
			// \033[2J -> clears the screen
			fmt.Print("\033[H\033[2J")
			str := run(fileName, readLineCount, searchTerm, *grepOnly)
			fmt.Printf("%s", str)
		}
	}
}

func run(fileName string, readLineCount int16, searchTerm string, grepOnly bool) string {
	var str string
	var err error
	if grepOnly {
		str, err = FindNSearchMatches(fileName, readLineCount, searchTerm)
	} else {
		str, err = SearchLastNLines(fileName, readLineCount, searchTerm)
	}

	if err != nil {
		fmt.Printf("err: %v\n", err)
		return err.Error()
	}
	return str
}

func FindNSearchMatches(fileName string, readLineCount int16, search string) (string, error) {
	searchTerm := []byte(search)

	f, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := stat.Size()
	if size == 0 {
		return "", nil
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return "", err
	}
	defer syscall.Munmap(data)

	highlightedSearch := append(
		append([]byte{}, highlightColor...),
		append(searchTerm, resetColor...)...,
	)

	predictedOffset := len(data) - TotalThroughput
	offset := max(0, predictedOffset)
	relevantData := data[offset:]

	if len(searchTerm) == 0 {
		countOffset := findOffset(relevantData, readLineCount)
		return string(relevantData[countOffset:]), nil
	}

	totalCount := int(readLineCount)
	end := len(relevantData)
	for totalCount > 0 && end > 0 {
		newStart := max(end-32_768, 0)
		totalCount -= bytes.Count(relevantData[newStart:end], searchTerm)
		end = newStart
	}
	relevantData = relevantData[end:]

	// this is needed for the compromise that the search replace miss is minimal
	// as we search replace on chunks it could happen that we don't have the needed bytes
	// in the chunk to match the search so we miss the replace
	neededWorkers := len(relevantData) / 32_768
	workers := runtime.NumCPU()
	chunks := splitTasks(relevantData, min(neededWorkers, workers))

	var wg sync.WaitGroup
	for i := range chunks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if len(searchTerm) != 0 && bytes.Contains(chunks[i], searchTerm) {
				chunks[i] = filterAndHighlightSearch(chunks[i], searchTerm, highlightedSearch)
			}
		}(i)
	}
	wg.Wait()

	// only present needed readLineCount
	result := bytes.Join(chunks, nil)
	countOffset := findOffset(result, readLineCount)
	return string(result[countOffset:]), nil
}

func SearchLastNLines(fileName string, readLineCount int16, search string) (string, error) {
	searchTerm := []byte(search)

	f, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := stat.Size()
	if size == 0 {
		return "", nil
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return "", err
	}
	defer syscall.Munmap(data)

	highlightedSearch := append(
		append([]byte{}, highlightColor...),
		append(searchTerm, resetColor...)...,
	)

	offset := findOffset(data, readLineCount)
	relevantData := data[offset:]

	if len(searchTerm) == 0 {
		return string(relevantData), nil
	}

	// this is needed for the compromise that the search replace miss is minimal
	// as we search replace on chunks it could happen that we don't have the needed bytes
	// in the chunk to match the search so we miss the replace
	neededWorkers := len(relevantData) / 32_768
	workers := runtime.NumCPU()
	chunks := splitTasks(relevantData, min(neededWorkers, workers))

	var wg sync.WaitGroup
	for i := range chunks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if len(searchTerm) != 0 && bytes.Contains(chunks[i], searchTerm) {
				chunks[i] = bytes.ReplaceAll(
					chunks[i],
					searchTerm,
					highlightedSearch,
				)
			}
		}(i)
	}
	wg.Wait()

	return string(bytes.Join(chunks, nil)), nil
}

func filterAndHighlightSearch(chunk, search, highlight []byte) []byte {
	res := make([]byte, 0, len(chunk))
	start := 0

	for i := 0; i < len(chunk); i++ {
		if chunk[i] != '\n' {
			continue
		}

		line := chunk[start : i+1]

		pos := bytes.Index(line, search)
		if pos == -1 {
			start = i + 1
			continue
		}

		// continue as long as search is found
		last := 0
		for pos != -1 {
			res = append(res, line[last:pos]...)
			res = append(res, highlight...)

			last = pos + len(search)
			pos = bytes.Index(line[last:], search)
			if pos != -1 {
				pos += last
			}
		}

		res = append(res, line[last:]...)
		start = i + 1
	}

	return res
}

func splitTasks[T any](tasks []T, n int) [][]T {
	var chunks [][]T
	length := len(tasks)
	if n <= 0 {
		n = 1
	}
	for i := 0; i < n; i++ {
		start := i * length / n
		end := (i + 1) * length / n
		chunks = append(chunks, tasks[start:end])
	}
	return chunks
}

// this is not much faster than a primitive one but it could also scale better
// with newer cpus cause of better use of ipc
// PERF: try stuff with Count as it uses bytecode simd
// newlineCnt += int16(bytes.Count(chunk, []byte{'\n'}))
func findOffset(data []byte, readLineCount int16) int {
	var newlineCount int16 = 0
	for i := len(data) - 1; i >= 4; i -= 5 {
		if data[i] == '\n' {
			newlineCount++
			if newlineCount == readLineCount {
				return i + 1
			}
		}
		if data[i-1] == '\n' {
			newlineCount++
			if newlineCount == readLineCount {
				return i
			}
		}
		if data[i-2] == '\n' {
			newlineCount++
			if newlineCount == readLineCount {
				return i - 1
			}
		}
		if data[i-3] == '\n' {
			newlineCount++
			if newlineCount == readLineCount {
				return i - 2
			}
		}
		if data[i-4] == '\n' {
			newlineCount++
			if newlineCount == readLineCount {
				return i - 3
			}
		}
	}

	for i := min(4, len(data)-1); i >= 0; i-- {
		if data[i] == '\n' {
			newlineCount++
			if newlineCount == readLineCount {
				return i + 1
			}
		}
	}
	return 0
}

func tailF(f *os.File, path string, tailCh chan<- string) {
	offset, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		log.Fatal(err)
	}
	reader := bufio.NewReader(f)
	for {
		// Try reading a line
		line, err := reader.ReadString('\n')
		if err == nil {
			tailCh <- line
			offset += int64(len(line))
			continue
		}

		// If nothing new: sleep and check again
		// TODO: try with lock
		time.Sleep(5000 * time.Millisecond)

		// Check if file grew
		stat, err := os.Stat(path)
		if err != nil {
			log.Fatal(err)
		}

		// // If file was truncated (e.g., rotated), reset
		if stat.Size() < offset {
			f.Close()
			f, _ = os.Open(path)
			reader = bufio.NewReader(f)
			offset = 0
			continue
		}
	}
}
