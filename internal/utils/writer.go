package utils

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// JSON 저장 함수 개선
func SaveToJSON(data interface{}, filename string) error {
	// 디렉토리가 없으면 생성
	dir := filepath.Dir(filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("디렉토리 생성 실패: %v", err)
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("JSON 변환 실패: %v", err)
	}

	// 파일이 이미 존재하는지 확인
	if _, err := os.Stat(filename); err == nil {
		log.Printf("⚠️ 파일이 이미 존재합니다: %s", filename)
	}

	if err := os.WriteFile(filename, jsonData, 0644); err != nil {
		return fmt.Errorf("파일 저장 실패: %v", err)
	}

	log.Printf("💾 저장 완료: %s (%d bytes)", filename, len(jsonData))
	return nil
}
