package main

import (
	"fmt"
	"log"
	"naverCrawler/internal/crawling"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

func main() {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		fmt.Println("Error loading .env file:", err)
		return
	}

	blogID := os.Getenv("NAVER_BLOG_ID")
	if blogID == "" {
		log.Fatal("NAVER_BLOG_ID 환경 변수가 설정되지 않았습니다.")
	}

	maxPages := 10
	if maxPagesStr := os.Getenv("MAX_PAGES"); maxPagesStr != "" {
		if v, err := strconv.Atoi(maxPagesStr); err == nil {
			maxPages = v
		} else {
			log.Printf("MAX_PAGES 환경변수 파싱 실패: %v, 기본값 10 사용", err)
		}
	}

	log.Printf("🎯 대상 블로그: %s", blogID)
	log.Printf("📄 크롤링 페이지 수: %d", maxPages)

	posts, err := crawling.CrawlBlog(blogID, maxPages)
	if err != nil {
		log.Fatal("❌ 크롤링 중 오류 발생:", err)
	}

	fmt.Printf("✅ 크롤링 완료! 총 %d개 블로그 게시글 수집\n", len(posts))
}
