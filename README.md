# Gosuda Website

[![Website](https://img.shields.io/badge/visit-gosuda.org-blue?style=flat-square)](https://gosuda.org)

This repository contains the source code for the Gosuda static website and blog. All content is written in Markdown and automatically processed through CI/CD for **translation and deployment**.

## 🖥️ Local Development

### Prerequisites
   - **golang** (1.25+)
   - **bun**
   - **LLM API Key** (for translation features)

### LLM API Configuration
```bash
# Google Vertex AI (default)
export LOCATION="us-central1"
export PROJECT_ID="your-gcp-project-id"

# Or Google AI Studio
export PROVIDER="aistudio"
export AI_STUDIO_API_KEY="your-key"

# Disable translation
export LLM_INIT="false"
```

### Build & Translate
   ```bash
   make build
   ```

### Start local server
   ```bash
   make run
   ```

## ✍️ Writing a new post

### 1. **Create a Markdown file in `/root/blog/`**  
   ```bash
   blog/my-new-post.md
   ```

### 2. **Add frontmatter metadata (at the top of the file)**  
   ```yaml
   ---
   author: <Your Name>
   title: <Post Title>
   ---
   ```

### 3. **Write your content in Markdown**

### 4. **Commit, Push, and Open a Pull Request**
   ```bash
   git add blog/my-new-post.md
   git commit -m "Add new blog post: my-new-post"
   git push origin my-branch
   ```