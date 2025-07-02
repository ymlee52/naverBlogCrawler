package crawling

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"naverCrawler/internal/utils"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BlogPost represents a blog post with only title, date and URL.
type BlogPost struct {
	Title       string `json:"title"`
	WriteDate   string `json:"write_date"`
	OriginalURL string `json:"url"`
}

// NaverBlogResponse represents the response from Naver Blog API
type NaverBlogResponse struct {
	ResultCode    string `json:"resultCode"`
	ResultMessage string `json:"resultMessage"`
	PostList      []struct {
		LogNo            string `json:"logNo"`
		Title            string `json:"title"`
		CategoryNo       string `json:"categoryNo"`
		ParentCategoryNo string `json:"parentCategoryNo"`
		CommentCount     string `json:"commentCount"`
		ReadCount        string `json:"readCount"`
		AddDate          string `json:"addDate"`
	} `json:"postList"`
	CountPerPage string `json:"countPerPage"`
	TotalCount   string `json:"totalCount"`
}

// 게시글 목록 가져오기 - 제목, 날짜, URL만
func GetBlogPostList(blogID string, page int) ([]BlogPost, error) {
	apiURL := fmt.Sprintf("https://blog.naver.com/PostTitleListAsync.naver?blogId=%s&viewdate=&currentPage=%d&categoryNo=25&parentCategoryNo=0&countPerPage=5", blogID, page)

	resp, err := client.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("게시글 목록 요청 실패: %v", err)
	}
	defer resp.Body.Close()

	// 응답 본문 읽기
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("응답 읽기 실패: %v", err)
	}

	// 작은따옴표를 큰따옴표로 변환
	jsonStr := strings.ReplaceAll(string(body), "'", "\"")

	var blogResponse NaverBlogResponse
	if err := json.Unmarshal([]byte(jsonStr), &blogResponse); err != nil {
		return nil, fmt.Errorf("JSON 파싱 실패: %v", err)
	}

	if blogResponse.ResultCode != "S" {
		return nil, fmt.Errorf("API 응답 오류: %s", blogResponse.ResultMessage)
	}

	var posts []BlogPost
	for _, post := range blogResponse.PostList {
		// 제목 URL 디코딩
		decodedTitle, err := url.QueryUnescape(post.Title)
		if err != nil {
			log.Printf("⚠️ 제목 디코딩 실패: %v, 원본 제목 사용", err)
			decodedTitle = post.Title
		}

		posts = append(posts, BlogPost{
			Title:       decodedTitle,
			WriteDate:   post.AddDate,
			OriginalURL: fmt.Sprintf("https://blog.naver.com/%s/%s", blogID, post.LogNo),
		})
	}

	if len(posts) == 0 {
		log.Printf("⚠️ 게시글을 찾을 수 없습니다. URL: %s", apiURL)
	}

	return posts, nil
}

// CrawlBlog performs the main crawling operation for a Naver blog - title, date and URL only
func CrawlBlog(blogID string, maxPages int) ([]BlogPost, error) {
	log.Printf("🚀 네이버 블로그 '%s' 크롤링 시작... (제목, 날짜, URL만)", blogID)

	outputDir := "output_blog"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("출력 디렉토리 생성 실패: %v", err)
	}

	var allPosts []BlogPost

	for page := 1; page <= maxPages; page++ {
		log.Printf("🔄 %d/%d 페이지 처리 중...", page, maxPages)

		postsOnPage, err := GetBlogPostList(blogID, page)
		if err != nil {
			log.Printf("⚠️ 페이지 %d 처리 실패: %v", page, err)
			continue
		}

		if len(postsOnPage) == 0 {
			log.Printf("⚠️ 페이지 %d에서 게시글을 찾을 수 없습니다", page)
			continue
		}

		allPosts = append(allPosts, postsOnPage...)
		log.Printf("✅ 페이지 %d에서 %d개 게시글 수집", page, len(postsOnPage))
	}

	if len(allPosts) > 0 {
		if err := saveFullResults(blogID, allPosts, outputDir); err != nil {
			log.Printf("⚠️ 전체 결과 저장 실패: %v", err)
		}
		printResults(allPosts)
	} else {
		fmt.Println("⚠️ 수집된 게시글이 없습니다. 블로그 ID를 확인해주세요.")
	}

	log.Printf("🎉 네이버 블로그 '%s' 크롤링 완료! 총 %d개 게시글 수집", blogID, len(allPosts))
	return allPosts, nil
}

func saveFullResults(blogID string, posts []BlogPost, outputDir string) error {
	timestamp := time.Now().Format("060102_150405")
	fullFilename := filepath.Join(outputDir, fmt.Sprintf("%s_%s.json", blogID, timestamp))

	formattedPosts := formatPosts(posts)
	if err := utils.SaveToJSON(formattedPosts, fullFilename); err != nil {
		return fmt.Errorf("전체 결과 저장 실패: %v", err)
	}

	return nil
}

func formatPosts(posts []BlogPost) []map[string]interface{} {
	var formattedPosts []map[string]interface{}
	for _, post := range posts {
		formattedPosts = append(formattedPosts, map[string]interface{}{
			"title":      post.Title,
			"write_date": post.WriteDate,
			"url":        post.OriginalURL,
		})
	}
	return formattedPosts
}

func printResults(posts []BlogPost) {
	fmt.Printf("\n📊 수집 결과 요약:\n")
	for i, post := range posts {
		if i >= 10 {
			fmt.Printf("... 외 %d개 게시글\n", len(posts)-10)
			break
		}
		fmt.Printf("📌 [%d] %s\n", i+1, post.Title)
		fmt.Printf("   📅 %s | 🔗 %s\n", post.WriteDate, post.OriginalURL)
		fmt.Println()
	}
}
