# Gosuda Website

[![Website](https://img.shields.io/badge/visit-gosuda.org-blue?style=flat-square)](https://gosuda.org)

This repository contains the source code for the Gosuda static website and blog. All content is written in Markdown and automatically processed through CI/CD for **translation and deployment**.

## 🖥️ Local Development

### Prerequisites
   - **golang** (1.25+)
   - **bun**
   - **LLM API Key** (for translation features)

### LLM API Configuration
This project uses Google's Gemini model for automatic translation. Choose one of the following methods:

#### Option 1: Google Vertex AI (Default)
Use Gemini model through Google Cloud Platform's Vertex AI:

```bash
# Set environment variables
export LOCATION="us-central1"  # or your preferred GCP region
export PROJECT_ID="your-gcp-project-id"

# Set up Google Cloud authentication (requires gcloud CLI)
gcloud auth application-default login
```

> **Note**: Vertex AI API must be enabled and you need appropriate permissions for the project.

#### Option 2: Google AI Studio (Alternative)
To use Google AI Studio API:

```bash
# Set environment variables
export PROVIDER="aistudio"
export AI_STUDIO_API_KEY="your-ai-studio-api-key"
```

Get your AI Studio API key from [Google AI Studio](https://aistudio.google.com/).

#### Skip LLM Initialization
To disable translation features:
```bash
export LLM_INIT="false"
```

> **Important**: Never commit API keys or credentials to git!

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