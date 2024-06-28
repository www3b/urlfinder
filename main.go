package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"sync"
)

func main() {
	// Define command-line flags
	urlListFile := flag.String("l", "", "File containing list of URLs")
	outputFile := flag.String("o", "", "Output file to write links")
	threadCount := flag.Int("t", 50, "Number of concurrent threads (default: 50)")
	flag.Parse()

	// Check if the required flags are provided
	if *urlListFile == "" || *outputFile == "" {
		fmt.Println("Usage: extractlinks -l urllist.txt -o outputfile.txt -t threadCount")
		return
	}

	// Read URLs from the file
	urls, err := readLines(*urlListFile)
	if err != nil {
		fmt.Printf("Error reading URL list file: %v\n", err)
		return
	}

	linkPattern := `https://[a-zA-Z0-9./?=_-]+`
	linkRegex := regexp.MustCompile(linkPattern)

	var wg sync.WaitGroup
	urlCh := make(chan string, len(urls))
	linkCh := make(chan []string)

	// Create worker pool
	for i := 0; i < *threadCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for url := range urlCh {
				resp, err := http.Get(url)
				if err != nil {
					fmt.Printf("Error fetching URL %s: %v\n", url, err)
					continue
				}
				defer resp.Body.Close()

				body, err := ioutil.ReadAll(resp.Body)
				if err != nil {
					fmt.Printf("Error reading response body for URL %s: %v\n", url, err)
					continue
				}

				foundLinks := linkRegex.FindAllString(string(body), -1)
				linkCh <- foundLinks
			}
		}()
	}

	// Send URLs to the channel
	go func() {
		for _, url := range urls {
			urlCh <- url
		}
		close(urlCh)
	}()

	// Collect links from workers and write to output file in real-time
	go func() {
		wg.Wait()
		close(linkCh)
	}()

	// Open the output file
	file, err := os.Create(*outputFile)
	if err != nil {
		fmt.Printf("Error creating output file: %v\n", err)
		return
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for links := range linkCh {
		for _, link := range links {
			_, err := writer.WriteString(link + "\n")
			if err != nil {
				fmt.Printf("Error writing to output file: %v\n", err)
				return
			}
		}
		writer.Flush()
	}

	fmt.Printf("Links successfully written to %s\n", *outputFile)
}

// readLines reads a file and returns the lines as a slice of strings
func readLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}
