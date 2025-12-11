package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: go run main.go <name>")
		return
	}
	// TODO: check if file exist -> if not return
	fileName := os.Args[len(os.Args)-1]
	fmt.Printf("last: %v\n", fileName)
	n := flag.Int("n", 10, "Number of lines")

	flag.Parse()
	fmt.Printf("n: %v\n", *n)
	fmt.Println("")
	str, err := ReadLastNLines(fileName, *n)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		return
	}
	fmt.Printf("str: \n%v\n", str)
}

func ReadLastNLines(fileName string, readLineCount int) (string, error) {
	fmt.Printf("fileName: %v\n", fileName)
	f, err := os.Open(fileName)
	if err != nil {
		return "", err
	}
	defer f.Close()

	const chunkSize = 1024
	stat, _ := f.Stat()
	size := stat.Size()
	fmt.Printf("size: %v\n", size)

	var buf []byte
	var readSize int64
	var newlineCnt int

	for size > 0 && newlineCnt <= readLineCount {
		readSize = min(size, chunkSize)
		size -= readSize

		chunk := make([]byte, readSize)
		_, err := f.ReadAt(chunk, size)
		if err != nil {
			return "", err
		}

		// print only as much lines as given readLineCount-> -n flag
		for i := len(chunk) - 1; i >= 0; i-- {
			if chunk[i] == '\n' {
				newlineCnt++
				if newlineCnt > readLineCount {
					buf = append(buf, chunk[i+1:]...)
					fmt.Printf("\"in first return\": %v\n", "in first return")
					return string(buf), nil
				}
			}
		}
		buf = append(buf, chunk...)
	}

	fmt.Printf("\"last return\": %v\n", "last return")
	return string(buf), nil
}
