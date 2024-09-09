package main

import (
	"net/http"
	"time"

	"github.com/a-h/templ"
	"gosuda.org/website/view"
)

func main() {
	blogPosts := []*view.BlogPostPreview{
		{
			Author:  "방장아님",
			Date:    time.Date(2024, 9, 3, 0, 0, 0, 0, time.UTC),
			Title:   "Scaling GoSuda's Infrastructure for Global Reach",
			Content: "In this post, we explore the challenges and solutions in scaling GoSuda's infrastructure to meet the demands of our growing global user base. Learn about our approach to distributed systems, load balancing, and data replication strategies.",
		},
		{
			Author:  "아Zig초보",
			Date:    time.Date(2024, 9, 5, 0, 0, 0, 0, time.UTC),
			Title:   "Developing Secure and Reliable APIs at GoSuda",
			Content: "Security and reliability are paramount in API development. This article delves into GoSuda's best practices for creating robust APIs, including authentication mechanisms, rate limiting, and comprehensive error handling to ensure a seamless developer experience.",
		},
		{
			Author:  "snowmerak",
			Date:    time.Date(2024, 9, 7, 0, 0, 0, 0, time.UTC),
			Title:   "Building a Culture of Innovation at GoSuda",
			Content: "Innovation is at the heart of GoSuda's success. In this post, we dive into the strategies and practices that foster a culture of continuous innovation within our organization. From encouraging creative thinking to implementing innovative ideas, discover how GoSuda stays at the forefront of technological advancements.",
		},
		{
			Author:  "GoSuda Team",
			Date:    time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC),
			Title:   "Introducing GoSuda's New AI-Powered Analytics Platform",
			Content: "We're excited to announce the launch of our new AI-powered analytics platform. This cutting-edge tool will revolutionize how businesses interpret and act on their data. Dive into the features and see how it can transform your decision-making process.",
		},
		{
			Author:  "TechGuru",
			Date:    time.Date(2024, 9, 12, 0, 0, 0, 0, time.UTC),
			Title:   "The Future of Cloud Computing: GoSuda's Perspective",
			Content: "Cloud computing is evolving rapidly. In this post, we share GoSuda's vision for the future of cloud technologies, including edge computing, serverless architectures, and AI-driven infrastructure management.",
		},
		{
			Author:  "CodeMaster",
			Date:    time.Date(2024, 9, 15, 0, 0, 0, 0, time.UTC),
			Title:   "Mastering Concurrency in Go: Tips from GoSuda Engineers",
			Content: "Concurrency is a powerful feature of Go, but it can be challenging to master. Our engineers share their top tips and best practices for writing efficient, bug-free concurrent code in Go.",
		},
	}

	featuredPosts := []view.FeaturedPost{
		{
			Title: "The Rise of Edge Computing: GoSuda's Innovative Approach",
			Link:  "#",
		},
		{
			Title: "How GoSuda is Revolutionizing Data Privacy",
			Link:  "#",
		},
		{
			Title: "GoSuda's Open Source Contributions: A Year in Review",
			Link:  "#",
		},
	}

	http.Handle("/", templ.Handler(view.GosudaBlog(blogPosts, featuredPosts)))

	http.ListenAndServe(":8080", nil)
}
