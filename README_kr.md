# Gosuda 웹사이트

[![Website](https://img.shields.io/badge/visit-gosuda.org-blue?style=flat-square)](https://gosuda.org)

이 저장소는 Gosuda 정적 웹사이트와 블로그의 소스 코드를 포함하고 있습니다. 모든 콘텐츠는 Markdown으로 작성되며 CI/CD를 통해 **자동 번역 및 배포**됩니다.

## 🖥️ 로컬 개발

### 필수 요구사항
   - **golang** (1.25+)
   - **bun**
   - **LLM API 키** (번역 기능용)

### LLM API 키 설정
이 프로젝트는 자동 번역을 위해 Google의 Gemini 모델을 사용합니다. 다음 중 하나의 방법으로 설정하세요:

#### 방법 1: Google Vertex AI 사용 (기본값)
Google Cloud Platform의 Vertex AI를 통해 Gemini 모델을 사용합니다:

```bash
# 환경 변수 설정
export LOCATION="us-central1"  # 또는 원하는 GCP 리전
export PROJECT_ID="your-gcp-project-id"

# Google Cloud 인증 설정 (gcloud CLI가 설치되어 있어야 함)
gcloud auth application-default login
```

> **참고**: Vertex AI API가 활성화되어 있고 프로젝트에 대한 적절한 권한이 있어야 합니다.

#### 방법 2: Google AI Studio 사용 (대안)
Google AI Studio API를 사용하려면:

```bash
# 환경 변수 설정
export PROVIDER="aistudio"
export AI_STUDIO_API_KEY="your-ai-studio-api-key"
```

AI Studio API 키는 [Google AI Studio](https://aistudio.google.com/)에서 발급받을 수 있습니다.

#### LLM 초기화 건너뛰기
번역 기능을 사용하지 않으려면:
```bash
export LLM_INIT="false"
```

> **중요**: API 키나 자격증명은 절대 git에 커밋하지 마세요!

### 빌드 및 번역
   ```bash
   make build
   ```

### 로컬 서버 시작
   ```bash
   make run
   ```
   서버 시작 후 `http://localhost:8080`에서 사이트를 확인할 수 있습니다.

## ✍️ 새 포스트 작성하기

### 1. **`/root/blog/` 디렉토리에 Markdown 파일 생성**  
   ```bash
   blog/my-new-post.md
   ```

### 2. **파일 상단에 프론트매터 메타데이터 추가**  
   ```yaml
   ---
   author: <작성자 이름>
   title: <포스트 제목>
   date: 2025-01-03  # 선택사항
   tags: [태그1, 태그2]  # 선택사항
   ---
   ```

### 3. **Markdown으로 콘텐츠 작성**
   일반적인 Markdown 문법을 사용하여 포스트를 작성하세요:
   - 제목: `#`, `##`, `###` 등
   - 목록: `-` 또는 `1.`
   - 코드 블록: \`\`\`언어명
   - 링크: `[텍스트](URL)`
   - 이미지: `![설명](이미지URL)`

### 4. **커밋, 푸시 및 Pull Request 생성**
   ```bash
   git add blog/my-new-post.md
   git commit -m "Add new blog post: my-new-post"
   git push origin my-branch
   ```
   
   GitHub에서 Pull Request를 생성하면 자동으로 빌드 및 번역이 진행됩니다.

## 🚀 배포 프로세스

1. **Pull Request 생성**: 새 포스트나 변경사항을 feature 브랜치에 커밋
2. **자동 번역**: CI/CD 파이프라인이 자동으로 콘텐츠를 다국어로 번역
3. **리뷰**: 번역된 콘텐츠와 변경사항 검토
4. **머지**: main 브랜치에 머지하면 자동으로 프로덕션에 배포

## 📁 프로젝트 구조

```
gosuda-website/
├── root/
│   └── blog/          # 블로그 포스트 (Markdown)
├── static/            # 정적 파일 (CSS, JS, 이미지)
├── templates/         # HTML 템플릿
├── Makefile          # 빌드 명령어
└── .env              # API 키 설정 (git에 포함되지 않음)
```

## 🔧 문제 해결

### 번역이 작동하지 않는 경우:
1. API 키가 올바르게 설정되었는지 확인
2. API 키에 충분한 크레딧이 있는지 확인
3. 네트워크 연결 상태 확인

### 빌드 오류가 발생하는 경우:
1. Go와 Bun이 올바르게 설치되었는지 확인
2. `go mod download` 실행하여 의존성 설치
3. `bun install` 실행하여 JavaScript 의존성 설치

## 📝 라이선스

이 프로젝트의 라이선스 정보는 [LICENSE](LICENSE) 파일을 참조하세요.

## 🤝 기여하기

기여를 환영합니다! 다음 단계를 따라주세요:

1. 이 저장소를 포크합니다
2. 새 브랜치를 생성합니다 (`git checkout -b feature/amazing-feature`)
3. 변경사항을 커밋합니다 (`git commit -m '멋진 기능 추가'`)
4. 브랜치에 푸시합니다 (`git push origin feature/amazing-feature`)
5. Pull Request를 생성합니다

질문이나 제안사항이 있으면 이슈를 생성해주세요!