# Gosuda 웹사이트

[![Website](https://img.shields.io/badge/visit-gosuda.org-blue?style=flat-square)](https://gosuda.org)

이 저장소는 Gosuda 정적 웹사이트와 블로그의 소스 코드를 포함하고 있습니다. 모든 콘텐츠는 Markdown으로 작성되며 CI/CD를 통해 **자동 번역 및 배포**됩니다.

## 🖥️ 로컬 개발

### 필수 요구사항
   - **golang** (1.25+)
   - **bun**
   - **LLM API 키** (번역 기능용)

### LLM API 설정
```bash
# Google Vertex AI (기본)
export LOCATION="us-central1"
export PROJECT_ID="your-gcp-project-id"

# 또는 Google AI Studio
export PROVIDER="aistudio"
export AI_STUDIO_API_KEY="your-key"

# 번역 비활성화
export LLM_INIT="false"
```

### 빌드 및 번역
   ```bash
   make build
   ```

### 로컬 서버 시작
   ```bash
   make run
   ```

## ✍️ 새 포스트 작성하기

### 1. **`/root/blog/`에 Markdown 파일 생성**  
   ```bash
   blog/my-new-post.md
   ```

### 2. **파일 상단에 프론트매터 메타데이터 추가**  
   ```yaml
   ---
   author: <작성자 이름>
   title: <포스트 제목>
   ---
   ```

### 3. **Markdown으로 콘텐츠 작성**

### 4. **커밋, 푸시 및 Pull Request 생성**
   ```bash
   git add blog/my-new-post.md
   git commit -m "Add new blog post: my-new-post"
   git push origin my-branch
   ```