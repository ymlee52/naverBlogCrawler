package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"naverCrawler/internal/crawling"
	"os"
	"strings"
	"sync"
	"time"
)

type BlogCrawlResult struct {
	URL   string `json:"url"`
	Body  string `json:"body,omitempty"`
	Error string `json:"error,omitempty"`
}

func main() {
	file, err := os.Open("urls.txt")
	if err != nil {
		log.Fatalf("urls.txt 파일 열기 실패: %v", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var urls []string
	for scanner.Scan() {
		url := scanner.Text()
		if url != "" {
			urls = append(urls, url)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Fatalf("파일 읽기 실패: %v", err)
	}

	var wg sync.WaitGroup
	concurrency := 10 // 동시에 실행할 고루틴 수
	sem := make(chan struct{}, concurrency)
	var mu sync.Mutex
	results := make([]BlogCrawlResult, len(urls))

	for i, url := range urls {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, url string) {
			defer wg.Done()
			defer func() { <-sem }()
			fmt.Printf("[%d/%d] %s 크롤링 중...\n", i+1, len(urls), url)
			content, err := crawling.CrawlBlogPostByURL(url)
			mu.Lock()
			if err != nil {
				results[i] = BlogCrawlResult{URL: url, Error: err.Error()}
			} else {
				flatContent := strings.ReplaceAll(content, "\r\n", "")
				flatContent = strings.ReplaceAll(flatContent, "\n", "")
				flatContent = strings.ReplaceAll(flatContent, "\r", "")
				results[i] = BlogCrawlResult{URL: url, Body: flatContent}
			}
			mu.Unlock()
		}(i, url)
	}
	wg.Wait()

	outputDir := "output_blog"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("output_blog 디렉토리 생성 실패: %v", err)
	}

	timestamp := time.Now().Format("060102_150405")
	jsonFile := fmt.Sprintf("%s/%s.json", outputDir, timestamp)
	f, err := os.Create(jsonFile)
	if err != nil {
		log.Fatalf("JSON 파일 생성 실패: %v", err)
	}
	defer f.Close()

	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(results); err != nil {
		log.Fatalf("JSON 인코딩 실패: %v", err)
	}

	fmt.Printf("모든 게시글 크롤링 완료! 결과는 %s에 저장되었습니다.\n", jsonFile)
}
