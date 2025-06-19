# 네이버 카페 크롤러

## 📝 프로젝트 설명
이 프로젝트는 네이버 카페의 게시글을 크롤링하는 Go 언어 기반의 크롤러입니다. 게시판 전체 크롤링과 키워드 검색 크롤링을 지원합니다.

## 🚀 시작하기

### 필수 조건
- Go 1.16 이상
- 네이버 카페 로그인 쿠키

### 설치
```bash
git clone https://github.com/yourusername/naverCafeCrawler.git
cd naverCafeCrawler
go mod download
```

### 환경 변수 설정
`.env` 파일을 생성하고 다음 내용을 입력하세요:

#### 게시판 크롤링용
```env
NAVER_CAFE_ID=your_cafe_id
NAVER_BOARD_ID=your_board_id
NAVER_COOKIE=your_cookie
```

#### 검색 크롤링용
```env
NAVER_CAFE_ID=your_cafe_id
NAVER_SEARCH_KEYWORD=검색할_키워드
NAVER_COOKIE=your_cookie
```

### 실행
```bash
# 검색 크롤링 실행
go run cmd/naverCafe/main.go
```

## 🔍 기능

### 1. 게시판 전체 크롤링
- 특정 게시판의 모든 게시글을 크롤링
- 게시판 ID를 지정하여 해당 게시판만 대상으로 함

### 2. 키워드 검색 크롤링 (새로운 기능!)
- 특정 키워드가 포함된 게시글만 크롤링
- 카페 전체에서 검색하여 관련 게시글을 찾음
- 더 정확하고 효율적인 데이터 수집 가능

## 💾 결과 저장
크롤링 결과는 `output` 폴더에 JSON 파일로 저장됩니다.

### 파일명 형식

#### 게시판 크롤링
- 전체 결과: `cafe_{카페ID}_board_{게시판ID}_{타임스탬프}_full.json`
- 페이지별 결과: `cafe_{카페ID}_board_{게시판ID}_{타임스탬프}_page_{페이지번호}.json`

#### 검색 크롤링
- 전체 결과: `cafe_{카페ID}_search_{검색키워드}_{타임스탬프}_full.json`
- 페이지별 결과: `cafe_{카페ID}_search_{검색키워드}_{타임스탬프}_page_{페이지번호}.json`

### JSON 구조
```json
[
  {
    "id": "게시글ID",
    "title": "제목",
    "writer": "작성자",
    "writer_level": "작성자 레벨",
    "is_staff": false,
    "is_manager": false,
    "write_date": "작성일시",
    "read_count": "조회수",
    "comment_count": "댓글수",
    "like_count": "좋아요수",
    "content": "게시글 내용 (HTML 형식)",
    "comments": [
      {
        "id": "댓글ID",
        "writer": "댓글 작성자",
        "writer_level": "댓글 작성자 레벨",
        "is_staff": false,
        "is_manager": false,
        "content": "댓글 내용",
        "write_date": "댓글 작성일시",
        "like_count": "댓글 좋아요수"
      }
    ]
  }
]
```

## 🔧 HTML 파싱 필요사항
현재 크롤러는 게시글 내용과 댓글을 HTML 형식으로 가져옵니다. 실제 사용을 위해서는 다음 작업이 필요합니다:

1. HTML 파싱 라이브러리 추가
   - `golang.org/x/net/html` 또는 `github.com/PuerkitoBio/goquery` 등의 라이브러리 사용 권장

2. 파싱 기능 구현
   - HTML 태그 제거
   - 특수 문자 처리
   - 이미지 URL 추출
   - 링크 처리
   - 이모지/이모티콘 처리

3. 파싱된 결과 저장
   - 원본 HTML과 파싱된 텍스트를 함께 저장
   - 이미지 URL 목록 별도 저장
   - 링크 정보 별도 저장

## ⚠️ 주의사항
- 네이버 카페의 이용약관을 준수하여 사용하세요.
- 과도한 요청은 IP 차단의 원인이 될 수 있습니다.
- 크롤링한 데이터의 저작권을 존중하세요.
- 검색 기능 사용 시 검색 결과가 없을 수 있습니다.

## 📄 라이선스
이 프로젝트는 MIT 라이선스 하에 배포됩니다. 