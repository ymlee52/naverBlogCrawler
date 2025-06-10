package crawling

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"naverCafeCrawler/internal/utils"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// BlogPost represents a blog post.
type BlogPost struct {
	ID          string        `json:"id"`
	Title       string        `json:"title"`
	Content     string        `json:"content"`
	Writer      string        `json:"writer"`
	WriteDate   string        `json:"write_date"`
	Comments    []BlogComment `json:"comments"`
	OriginalURL string        `json:"original_url"`
}

// BlogComment represents a comment on a blog post.
type BlogComment struct {
	ID        string `json:"id"`
	Content   string `json:"content"`
	Writer    string `json:"writer"`
	WriteDate string `json:"write_date"`
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

// 셀렉터 상수 정의
const (
	writerSelectors         = ".nick_name, .blog_author .author_name, .author, .writer, .nickname, .blog_name, .blog_name, .nickname"
	dateSelectors           = ".se_time, .blog_header_info .date, ._postContents .post_info .date, .post_date, .date, .write_date, .se_publishDate, .date"
	contentSelectors        = ".se-main-container, .post_content, .se-component.se-text.se-section, .sect_dsc, .post_ct, #content-area .post_content, .se-module-text, .pcol1 .post_content, .se-main-container, .post-view"
	commentSelectors        = ".comment_area .comment_item, ._commentWrapper .comment_row, .comment_list .comment, .cmt_area .cmt_item, .comment_item"
	commentContentSelectors = ".comment_text, .text_comment, .cmt_text, .comment_text_box"
	commentWriterSelectors  = ".comment_nick, .author_name, .cmt_nick, .comment_nick_box"
	commentDateSelectors    = ".comment_date, .date, .cmt_date, .comment_date_box"
)

// 게시글 목록 가져오기 - 개선된 버전
func GetBlogPostList(blogID string, page int) ([]BlogPost, error) {
	url := fmt.Sprintf("https://blog.naver.com/PostTitleListAsync.naver?blogId=%s&viewdate=&currentPage=%d&categoryNo=0&parentCategoryNo=&countPerPage=5", blogID, page)

	resp, err := client.Get(url)
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
		posts = append(posts, BlogPost{
			ID:          post.LogNo,
			Title:       post.Title,
			WriteDate:   post.AddDate,
			OriginalURL: fmt.Sprintf("https://blog.naver.com/%s/%s", blogID, post.LogNo),
		})
	}

	if len(posts) == 0 {
		log.Printf("⚠️ 게시글을 찾을 수 없습니다. URL: %s", url)
	}

	return posts, nil
}

// 게시글 상세 정보 가져오기 - 개선된 버전
func GetBlogPostDetail(blogID string, articleID string) (BlogPost, error) {
	url := fmt.Sprintf("https://blog.naver.com/PostView.naver?blogId=%s&logNo=%s", blogID, articleID)

	resp, err := client.Get(url)
	if err != nil {
		return BlogPost{}, fmt.Errorf("게시글 상세 로드 실패: %v", err)
	}
	defer resp.Body.Close()

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return BlogPost{}, fmt.Errorf("HTML 파싱 실패: %v", err)
	}

	// script 태그 제거
	doc.Find("script").Remove()

	// title 태그에서 제목 추출
	title := doc.Find("title").Text()
	// 네이버 블로그 제목에서 불필요한 부분 제거 (예: " : 네이버 블로그")
	title = strings.Split(title, " : 네이버 블로그")[0]

	// .se-main-container 내의 콘텐츠만 추출
	content := doc.Find(".se-main-container").Text()

	// 연속된 공백과 줄바꿈 정리
	content = strings.Join(strings.Fields(content), " ")

	// 불필요한 공백 제거
	content = strings.TrimSpace(content)

	blogPost := BlogPost{
		ID:          articleID,
		OriginalURL: url,
		Title:       title,
		Writer:      utils.FindFirstMatch(doc, writerSelectors),
		WriteDate:   utils.FindFirstMatch(doc, dateSelectors),
		Content:     content,
		Comments:    extractComments(doc),
	}

	if blogPost.Title == "" && blogPost.Content == "" {
		return blogPost, fmt.Errorf("게시글 정보를 추출할 수 없습니다")
	}

	return blogPost, nil
}

// CrawlBlog performs the main crawling operation for a Naver blog
func CrawlBlog(blogID string, maxPages int) ([]BlogPost, error) {
	log.Printf("🚀 네이버 블로그 '%s' 크롤링 시작...", blogID)

	outputDir := "output_blog"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("출력 디렉토리 생성 실패: %v", err)
	}

	var allPosts []BlogPost
	var mu sync.Mutex

	for page := 1; page <= maxPages; page++ {
		detailedPostsOnPage, err := processPage(blogID, page, maxPages)
		if err != nil {
			log.Printf("⚠️ 페이지 %d 처리 실패: %v", page, err)
			continue
		}

		if len(detailedPostsOnPage) == 0 {
			continue
		}

		mu.Lock()
		allPosts = append(allPosts, detailedPostsOnPage...)
		mu.Unlock()

		if err := savePageResults(blogID, page, detailedPostsOnPage, outputDir); err != nil {
			log.Printf("⚠️ 페이지 %d 결과 저장 실패: %v", page, err)
		}
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

// Helper functions

func extractComments(doc *goquery.Document) []BlogComment {
	var comments []BlogComment
	doc.Find(commentSelectors).Each(func(i int, s *goquery.Selection) {
		content := utils.CleanText(s.Find(commentContentSelectors).First().Text())
		if content != "" {
			comments = append(comments, BlogComment{
				ID:        fmt.Sprintf("%d", i+1),
				Content:   content,
				Writer:    utils.CleanText(s.Find(commentWriterSelectors).First().Text()),
				WriteDate: utils.CleanText(s.Find(commentDateSelectors).First().Text()),
			})
		}
	})
	return comments
}

func processPage(blogID string, page, maxPages int) ([]BlogPost, error) {
	log.Printf("🔄 %d/%d 페이지 처리 중...", page, maxPages)

	postsOnPage, err := GetBlogPostList(blogID, page)
	if err != nil {
		return nil, fmt.Errorf("게시글 목록 가져오기 실패: %v", err)
	}

	if len(postsOnPage) == 0 {
		return nil, fmt.Errorf("게시글을 찾을 수 없습니다")
	}

	var detailedPostsOnPage []BlogPost
	for i, post := range postsOnPage {
		log.Printf("  📖 %d페이지 게시글 %d/%d 상세 정보 처리 중... (ID: %s)", page, i+1, len(postsOnPage), post.ID)

		detail, err := GetBlogPostDetail(blogID, post.ID)
		if err != nil {
			log.Printf("⚠️ 게시글 %s 상세 정보 가져오기 실패: %v", post.ID, err)
			continue
		}

		if detail.Title != "" || detail.Content != "" {
			detailedPostsOnPage = append(detailedPostsOnPage, detail)
		}
	}

	return detailedPostsOnPage, nil
}

func savePageResults(blogID string, page int, posts []BlogPost, outputDir string) error {
	timestamp := time.Now().Format("20060102_150405")
	pageFilename := filepath.Join(outputDir, fmt.Sprintf("blog_%s_page_%d_%s.json", blogID, page, timestamp))

	formattedPosts := formatPosts(posts)
	if err := utils.SaveToJSON(formattedPosts, pageFilename); err != nil {
		return fmt.Errorf("페이지 결과 저장 실패: %v", err)
	}

	return nil
}

func saveFullResults(blogID string, posts []BlogPost, outputDir string) error {
	timestamp := time.Now().Format("20060102_150405")
	fullFilename := filepath.Join(outputDir, fmt.Sprintf("blog_%s_full_%s.json", blogID, timestamp))

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
			"title":   post.Title,
			"content": post.Content,
			"metadata": map[string]interface{}{
				"id":         post.ID,
				"writer":     post.Writer,
				"write_date": post.WriteDate,
				"url":        post.OriginalURL,
			},
			"comments": post.Comments,
		})
	}
	return formattedPosts
}

func printResults(posts []BlogPost) {
	fmt.Printf("\n📊 수집 결과 요약:\n")
	for i, post := range posts {
		if i >= 5 {
			fmt.Printf("... 외 %d개 게시글\n", len(posts)-5)
			break
		}
		fmt.Printf("📌 [%d] %s\n", i+1, post.Title)
		fmt.Printf("   👤 %s | 📅 %s | 💬 %d개 댓글\n", post.Writer, post.WriteDate, len(post.Comments))
		fmt.Printf("   📝 %s...\n", utils.TruncateString(post.Content, 100))
		fmt.Println()
	}
}
