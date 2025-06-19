package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"re_naverBlogCrawler/internal/crawling"

	"github.com/joho/godotenv"
)

func saveToJSON(data interface{}, filename string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 변환 실패: %v", err)
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("파일 저장 실패: %v", err)
	}

	return nil
}

func main() {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		fmt.Println("Error loading .env file:", err)
		return
	}

	cafeId := os.Getenv("NAVER_CAFE_ID") // 네이버 카페 ID 입력
	cookie := os.Getenv("NAVER_COOKIE")  // 환경 변수에서 쿠키 가져오기
	if cookie == "" {
		log.Fatal("NAVER_COOKIE 환경 변수가 설정되지 않았습니다.")
	}

	// 검색 키워드 가져오기
	keyword := os.Getenv("NAVER_SEARCH_KEYWORD")
	if keyword == "" {
		log.Fatal("NAVER_SEARCH_KEYWORD 환경 변수가 설정되지 않았습니다.")
	}

	// 최대 페이지 수 설정 (0은 무제한)
	maxPages := 10
	// pageSize 설정 (기본값: 10)
	pageSize := 15

	fmt.Printf("🔍 검색어 '%s'로 네이버 카페 크롤링 시작...\n", keyword)
	posts, err := crawling.CrawlSearchResults(cafeId, keyword, cookie, maxPages, pageSize)
	if err != nil {
		log.Fatal("❌ 크롤링 중 오류 발생:", err)
	}

	if len(posts) == 0 {
		fmt.Printf("⚠️ 검색어 '%s'에 대한 결과가 없습니다.\n", keyword)
		return
	}

	fmt.Printf("✅ 크롤링 완료! 총 %d개 게시글 수집\n", len(posts))

	// 콘솔에도 결과 출력
	for _, post := range posts {
		fmt.Printf("\n📌 [%d] %s\n", post["id"], post["title"])
		fmt.Printf("👤 작성자: %s (레벨: %s)\n", post["writer"], post["writer_level"])
		fmt.Printf("📅 작성일: %s\n", post["write_date"])
		fmt.Printf("📊 조회수: %d, 댓글: %d, 좋아요: %d\n", post["read_count"], post["comment_count"], post["like_count"])

		// 게시글 내용 출력
		if content, ok := post["content"].(string); ok {
			fmt.Printf("\n📝 내용:\n%s\n", content)
		}

		// 댓글 출력
		if comments, ok := post["comments"].([]map[string]interface{}); ok && len(comments) > 0 {
			fmt.Printf("\n💬 댓글 (%d개):\n", len(comments))
			for _, comment := range comments {
				fmt.Printf("  - [%s] %s (%s)\n",
					comment["writer"],
					comment["content"],
					comment["write_date"])
			}
		}
		fmt.Println("\n" + strings.Repeat("─", 80)) // 구분선
	}
}
