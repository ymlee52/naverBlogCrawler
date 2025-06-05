# naverBlogCrawler

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

// Article represents a blog post.
type BlogPost struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Content     string        `json:"content"`
	Writer      string        `json:"writer"`
	WriteDate   string        `json:"write_date"`
	Comments    []BlogComment `json:"comments"`
	OriginalURL string        `json:"original_url"`
	// Add more fields as needed (e.g., tags, categories, view count if available)
}

// BlogComment represents a comment on a blog post.
type BlogComment struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Writer    string `json:"writer"`
	WriteDate string `json:"write_date"`
}

// HTTP 클라이언트 설정 (재사용)
var client = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	},
	Timeout: 15 * time.Second, // 블로그는 더 복잡할 수 있으므로 타임아웃을 조금 늘림
}

// 요청 간 랜덤 지연을 위한 함수 (재사용)
func randomSleep() {
	sleepTime := time.Duration(rand.Intn(3000)+1000) * time.Millisecond // 1초에서 4초 사이
	time.Sleep(sleepTime)
}

// HTTP 요청 보내고 응답 반환하는 함수 (재사용)
func getHTMLResponse(url string) (*goquery.Document, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// 일반적인 브라우저 User-Agent 사용
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.127 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")

	randomSleep()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP 오류: %d - %s", resp.StatusCode, url)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("HTML 파싱 실패: %v", err)
	}

	return doc, nil
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
}

// 게시글 목록 가져오기 (블로그에 맞게 수정)
// 네이버 블로그는 페이지마다 HTML 구조가 다르거나, 게시글 목록을 AJAX로 로드하는 경우가 많으므로
// 이 함수는 특정 블로그의 "전체보기" 또는 "목록" 페이지를 기준으로 작성됩니다.
// 실제 블로그 구조에 따라 셀렉터는 변경될 수 있습니다.
func getBlogPostList(blogID string, page int) ([]BlogPost, error) {
	// 네이버 블로그의 새로운 목록 URL 형식 사용
	listURL := fmt.Sprintf("https://blog.naver.com/PostList.naver?blogId=%s&currentPage=%d", blogID, page)

	fmt.Printf("\n[DEBUG] 크롤링 URL: %s\n", listURL)

	doc, err := getHTMLResponse(listURL)
	if err != nil {
		return nil, fmt.Errorf("블로그 목록 페이지 로드 실패 (%s): %v", listURL, err)
	}

	var posts []BlogPost

	// 게시글 목록 찾기 (새로운 셀렉터 사용)
	doc.Find("#PostListBody .post_item, .blog2_post, .post").Each(func(i int, s *goquery.Selection) {
		// 링크와 제목 찾기
		var href, title string
		linkEl := s.Find("a.post_link, .title_link, .blog2_title a").First()

		href, exists := linkEl.Attr("href")
		if !exists {
			fmt.Printf("[DEBUG] 링크를 찾을 수 없음: %s\n", s.Text())
			return
		}

		fmt.Printf("[DEBUG] 발견된 링크: %s\n", href)

		// 게시글 ID 추출
		articleID := ""
		if strings.Contains(href, "logNo=") {
			parts := strings.Split(href, "logNo=")
			if len(parts) > 1 {
				articleID = parts[1]
				if idx := strings.Index(articleID, "&"); idx != -1 {
					articleID = articleID[:idx]
				}
			}
		}

		if articleID == "" {
			fmt.Printf("[DEBUG] 게시글 ID를 찾을 수 없음: %s\n", href)
			return
		}

		// 제목 추출
		title = strings.TrimSpace(linkEl.Text())
		if title == "" {
			title = strings.TrimSpace(s.Find(".title_text, .se-title-text").First().Text())
		}

		if title == "" {
			fmt.Printf("[DEBUG] 제목을 찾을 수 없음: %s\n", href)
			return
		}

		post := BlogPost{
			ID:          articleID,
			Title:       title,
			OriginalURL: fmt.Sprintf("https://blog.naver.com/%s/%s", blogID, articleID),
		}

		fmt.Printf("[DEBUG] 발견된 게시글: ID=%s, 제목=%s\n", articleID, title)
		posts = append(posts, post)
	})

	log.Printf("✅ %s 블로그 %d페이지에서 %d개 게시글 발견", blogID, page, len(posts))
	return posts, nil
}

// 게시글 상세 정보 가져오기 (블로그에 맞게 수정)
// 네이버 블로그 게시글 페이지의 HTML 구조를 분석하여 내용을 추출합니다.
// 이 셀렉터는 예시이며, 실제 블로그마다 다릅니다.
func getBlogPostDetail(blogID string, articleID string) (BlogPost, error) {
	url := fmt.Sprintf("https://blog.naver.com/%s/%s", blogID, articleID)

	log.Printf("  - 게시글 상세 정보 수집 중: %s", url)
	doc, err := getHTMLResponse(url)
	if err != nil {
		return BlogPost{}, fmt.Errorf("블로그 게시글 상세 페이지 로드 실패 (%s): %v", url, err)
	}

	var blogPost BlogPost
	blogPost.ID = articleID
	blogPost.OriginalURL = url

	// 제목 추출
	blogPost.Title = strings.TrimSpace(doc.Find(".se-title-text, .tit_area .tit").First().Text())

	// 작성자 추출
	blogPost.Writer = strings.TrimSpace(doc.Find(".nick_name, .blog_author .author_name").First().Text())

	// 작성일 추출 (다양한 클래스가 있을 수 있음)
	writeDateSelector := ".se_time, .blog_header_info .date, ._postContents .post_info .date"
	blogPost.WriteDate = strings.TrimSpace(doc.Find(writeDateSelector).First().Text())
	// 필요시 날짜 형식을 파싱하여 표준화할 수 있습니다.

	// 본문 내용 추출
	// 네이버 블로그는 SmartEditor 또는 구형 에디터에 따라 클래스가 다를 수 있습니다.
	// .se-main-container (SmartEditor One), .post_content (구형), .se-component.se-text.se-section (SmartEditor 내부 텍스트)
	var contentBuilder strings.Builder
	doc.Find(".se-main-container, .post_content, .se-component.se-text.se-section, .sect_dsc").Each(func(i int, s *goquery.Selection) {
		// 이미지, 스티커, 기타 비텍스트 요소는 제외하고 텍스트만 추출
		s.Find("img, .se-sticker, .se-module-oglink, .se-map-container, .se-file-block").Remove()
		contentBuilder.WriteString(strings.TrimSpace(s.Text()) + "\n")
	})
	blogPost.Content = strings.TrimSpace(contentBuilder.String())

	// 댓글 추출
	var comments []BlogComment
	// 댓글은 보통 iframe에 있거나 동적으로 로드될 수 있습니다.
	// 여기서는 페이지에 직접 렌더링된 댓글을 가정합니다.
	// 실제 블로그의 댓글 영역 클래스를 확인해야 합니다.
	doc.Find(".comment_area .comment_item, ._commentWrapper .comment_row").Each(func(i int, s *goquery.Selection) {
		commentContent := strings.TrimSpace(s.Find(".comment_text, .text_comment").First().Text())
		commentWriter := strings.TrimSpace(s.Find(".comment_nick, .author_name").First().Text())
		commentDate := strings.TrimSpace(s.Find(".comment_date, .date").First().Text()) // 댓글 작성일
		if commentContent != "" {
			comments = append(comments, BlogComment{
				Content:   commentContent,
				Writer:    commentWriter,
				WriteDate: commentDate,
			})
		}
	})
	blogPost.Comments = comments

	log.Printf("  ✅ 게시글 %s 상세 정보 수집 완료 (제목: %s, 댓글 %d개)", articleID, blogPost.Title, len(comments))
	return blogPost, nil
}

// JSON 저장 함수 (재사용)
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

// 블로그 크롤링 메인 함수
func CrawlBlog(blogID string, maxPages int) ([]BlogPost, error) {
	log.Printf("🚀 네이버 블로그 '%s' 크롤링 시작...", blogID)

	var allPosts []BlogPost
	var mu sync.Mutex // 공유 데이터 (allPosts) 보호를 위한 뮤텍스

	outputDir := "output_blog"
	aiReadyDir := filepath.Join(outputDir, "ai_ready")

	// 출력 디렉토리 생성
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Printf("⚠️ 출력 디렉토리 생성 실패: %v", err)
	}
	if err := os.MkdirAll(aiReadyDir, 0755); err != nil {
		log.Printf("⚠️ AI Ready 디렉토리 생성 실패: %v", err)
	}

	// 첫 페이지부터 maxPages까지 크롤링
	ctx := context.Background()
	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(3) // 동시 처리 제한 (동시에 3개의 페이지를 크롤링)

	for page := 1; page <= maxPages; page++ {
		page := page // 클로저에서 올바른 'page' 값을 사용하기 위해 캡처
		eg.Go(func() error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				postsOnPage, err := getBlogPostList(blogID, page)
				if err != nil {
					return fmt.Errorf("페이지 %d 게시글 목록 가져오기 실패: %v", page, err)
				}

				var detailedPostsOnPage []BlogPost
				for i, post := range postsOnPage {
					log.Printf("  - %d페이지 게시글 %d/%d 상세 정보 처리 중...", page, i+1, len(postsOnPage))
					detail, err := getBlogPostDetail(blogID, post.ID)
					if err != nil {
						log.Printf("⚠️ 게시글 %s 상세 정보 가져오기 실패: %v", post.ID, err)
						continue // 상세 정보 가져오기 실패해도 다른 게시글 계속 진행
					}
					detailedPostsOnPage = append(detailedPostsOnPage, detail)
				}

				mu.Lock()
				allPosts = append(allPosts, detailedPostsOnPage...)
				mu.Unlock()

				// 페이지별로 파일 저장
				timestamp := time.Now().Format("20060102_150405")
				pageFilename := filepath.Join(outputDir, fmt.Sprintf("blog_%s_page_%d_%s.json", blogID, page, timestamp))
				if err := saveToJSON(detailedPostsOnPage, pageFilename); err != nil {
					log.Printf("⚠️ %d페이지 결과 저장 실패: %v", page, err)
				} else {
					log.Printf("💾 %d페이지 결과가 %s 파일로 저장되었습니다.", page, pageFilename)
				}

				// AI Ready 형식으로 데이터 변환 및 저장
				var aiReadyPosts []map[string]interface{}
				for _, post := range detailedPostsOnPage {
					aiReadyPost := map[string]interface{}{
						"title":   post.Title,
						"content": post.Content,
						"metadata": map[string]interface{}{
							"id":         post.ID,
							"writer":     post.Writer,
							"write_date": post.WriteDate,
							"url":        post.OriginalURL,
						},
						"comments": post.Comments,
					}
					aiReadyPosts = append(aiReadyPosts, aiReadyPost)
				}

				aiReadyFilename := filepath.Join(aiReadyDir, fmt.Sprintf("blog_%s_page_%d_%s_ai_ready.json",
					blogID, page, timestamp))
				if err := saveToJSON(aiReadyPosts, aiReadyFilename); err != nil {
					log.Printf("⚠️ %d페이지 AI Ready 결과 저장 실패: %v", page, err)
				} else {
					log.Printf("💾 %d페이지 AI Ready 결과가 %s 파일로 저장되었습니다.", page, aiReadyFilename)
				}

				log.Printf("✅ %d/%d 페이지 크롤링 완료 (수집 게시글 %d개)", page, maxPages, len(detailedPostsOnPage))
				return nil
			}
		})
	}

	// 모든 고루틴이 완료될 때까지 대기
	err := eg.Wait()
	if err != nil {
		return nil, err
	}

	// 전체 결과를 하나의 파일로 저장 (선택 사항, 필요 시 활성화)
	timestamp := time.Now().Format("20060102_150405")
	fullFilename := filepath.Join(outputDir, fmt.Sprintf("blog_%s_full_%s.json", blogID, timestamp))
	if err := saveToJSON(allPosts, fullFilename); err != nil {
		log.Printf("⚠️ 전체 결과 저장 실패: %v", err)
	} else {
		log.Printf("💾 전체 결과가 %s 파일로 저장되었습니다. (총 %d개 게시글)", fullFilename, len(allPosts))
	}

	// AI Ready 전체 결과도 저장
	var allAiReadyPosts []map[string]interface{}
	for _, post := range allPosts {
		aiReadyPost := map[string]interface{}{
			"title":   post.Title,
			"content": post.Content,
			"metadata": map[string]interface{}{
				"id":         post.ID,
				"writer":     post.Writer,
				"write_date": post.WriteDate,
				"url":        post.OriginalURL,
			},
			"comments": post.Comments,
		}
		allAiReadyPosts = append(allAiReadyPosts, aiReadyPost)
	}

	aiReadyFullFilename := filepath.Join(aiReadyDir, fmt.Sprintf("blog_%s_full_%s_ai_ready.json",
		blogID, timestamp))
	if err := saveToJSON(allAiReadyPosts, aiReadyFullFilename); err != nil {
		log.Printf("⚠️ AI Ready 전체 결과 저장 실패: %v", err)
	} else {
		log.Printf("💾 AI Ready 전체 결과가 %s 파일로 저장되었습니다. (총 %d개 게시글)", aiReadyFullFilename, len(allAiReadyPosts))
	}

	log.Printf("🎉 네이버 블로그 '%s' 크롤링 완료! 총 %d개 게시글 수집", blogID, len(allPosts))
	return allPosts, nil
}

func main() {
	err := godotenv.Load()
	if err != nil && !os.IsNotExist(err) {
		fmt.Println("Error loading .env file:", err)
		return
	}

	blogID := os.Getenv("NAVER_BLOG_ID") // 크롤링할 블로그 ID (예: 'blogID' from blog.naver.com/blogID)
	if blogID == "" {
		log.Fatal("NAVER_BLOG_ID 환경 변수가 설정되지 않았습니다.")
	}

	maxPages := 3 // 크롤링할 최대 페이지 수 (필요에 따라 조절)

	posts, err := CrawlBlog(blogID, maxPages)
	if err != nil {
		log.Fatal("❌ 크롤링 중 오류 발생:", err)
	}

	fmt.Printf("✅ 크롤링 완료! 총 %d개 블로그 게시글 수집\n", len(posts))

	// 콘솔에도 결과 출력 (간략하게)
	for _, post := range posts {
		fmt.Printf("\n📌 [ID: %s] %s\n", post.ID, post.Title)
		fmt.Printf("👤 작성자: %s\n", post.Writer)
		fmt.Printf("📅 작성일: %s\n", post.WriteDate)
		fmt.Printf("🔗 URL: %s\n", post.OriginalURL)
		fmt.Printf("📝 내용 요약:\n%s...\n", truncateString(post.Content, 200))

		if len(post.Comments) > 0 {
			fmt.Printf("💬 댓글 (%d개):\n", len(post.Comments))
			for i, comment := range post.Comments {
				if i >= 3 { // 최대 3개 댓글만 출력
					fmt.Printf("  ...외 %d개\n", len(post.Comments)-3)
					break
				}
				fmt.Printf("  - [%s] %s (%s)\n", comment.Writer, truncateString(comment.Content, 50), comment.WriteDate)
			}
		}
		fmt.Println("\n" + strings.Repeat("─", 80)) // 구분선
	}
}

// 문자열을 특정 길이로 자르고 "..."을 추가하는 헬퍼 함수
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
