package utils

import (
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// 문자열 정리 함수 개선
func CleanText(text string) string {
	if text == "" {
		return ""
	}

	// 탭, 개행 등을 공백으로 변환
	text = strings.ReplaceAll(text, "\t", " ")
	text = strings.ReplaceAll(text, "\n", " ")
	text = strings.ReplaceAll(text, "\r", " ")

	// 연속된 공백을 하나로
	text = strings.Join(strings.Fields(text), " ")

	// 앞뒤 공백 제거
	text = strings.TrimSpace(text)

	return text
}

// 문자열을 특정 길이로 자르고 "..."을 추가하는 헬퍼 함수
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// 여러 셀렉터 중 첫 번째 매칭을 찾는 함수
func FindFirstMatch(doc *goquery.Document, selectors string) string {
	return CleanText(doc.Find(selectors).First().Text())
}
