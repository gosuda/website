import { useState, useEffect } from 'react'
import { Search, Github } from 'lucide-react'

const blogPosts = [
  {
    author: '방장아님',
    date: 'September 3, 2024',
    title: "Scaling GoSuda's Infrastructure for Global Reach",
    content: "In this post, we explore the challenges and solutions in scaling GoSuda's infrastructure to meet the demands of our growing global user base. Learn about our approach to distributed systems, load balancing, and data replication strategies.",
  },
  {
    author: '아Zig초보',
    date: 'September 5, 2024',
    title: 'Developing Secure and Reliable APIs at GoSuda',
    content: "Security and reliability are paramount in API development. This article delves into GoSuda's best practices for creating robust APIs, including authentication mechanisms, rate limiting, and comprehensive error handling to ensure a seamless developer experience.",
  },
  {
    author: 'snowmerak',
    date: 'September 7, 2024',
    title: 'Building a Culture of Innovation at GoSuda',
    content: 'Innovation is at the heart of GoSuda's success. In this post, we dive into the strategies and practices that foster a culture of continuous innovation within our organization. From encouraging creative thinking to implementing innovative ideas, discover how GoSuda stays at the forefront of technological advancements.',
  },
  {
    author: 'GoSuda Team',
    date: 'September 10, 2024',
    title: 'Introducing GoSuda's New AI-Powered Analytics Platform',
    content: 'We're excited to announce the launch of our new AI-powered analytics platform. This cutting-edge tool will revolutionize how businesses interpret and act on their data. Dive into the features and see how it can transform your decision-making process.',
  },
  {
    author: 'TechGuru',
    date: 'September 12, 2024',
    title: 'The Future of Cloud Computing: GoSuda's Perspective',
    content: 'Cloud computing is evolving rapidly. In this post, we share GoSuda's vision for the future of cloud technologies, including edge computing, serverless architectures, and AI-driven infrastructure management.',
  },
  {
    author: 'CodeMaster',
    date: 'September 15, 2024',
    title: 'Mastering Concurrency in Go: Tips from GoSuda Engineers',
    content: 'Concurrency is a powerful feature of Go, but it can be challenging to master. Our engineers share their top tips and best practices for writing efficient, bug-free concurrent code in Go.',
  },
]

const featuredPosts = [
  {
    title: "The Rise of Edge Computing: GoSuda's Innovative Approach",
    link: "#",
  },
  {
    title: "How GoSuda is Revolutionizing Data Privacy",
    link: "#",
  },
  {
    title: "GoSuda's Open Source Contributions: A Year in Review",
    link: "#",
  },
]

export default function GosudaBlog() {
  const [isSearchOpen, setIsSearchOpen] = useState(false)

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === '/' && !isSearchOpen) {
        e.preventDefault()
        setIsSearchOpen(true)
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [isSearchOpen])

  const openCommandPalette = () => setIsSearchOpen(true)
  const closeCommandPalette = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      setIsSearchOpen(false)
    }
  }

  return (
    <div className="max-w-6xl mx-auto p-4 min-h-screen flex flex-col">
      <header className="flex justify-between items-center p-4 border-2 border-black rounded-lg mb-6">
        <h1 className="text-2xl font-bold">GoSuda Blog</h1>
        <nav className="flex items-center">
          <button onClick={openCommandPalette} className="mr-4 flex items-center">
            <Search className="w-5 h-5 mr-1" />
            Search
          </button>
          <a href="https://github.com/gosuda" target="_blank" rel="noopener noreferrer" className="flex items-center">
            <Github className="w-5 h-5 mr-1" />
            @github
          </a>
        </nav>
      </header>

      {isSearchOpen && (
        <div
          className="fixed inset-0 bg-black bg-opacity-50 flex justify-center items-center z-50"
          onClick={closeCommandPalette}
        >
          <div className="bg-white p-6 rounded-lg w-full max-w-2xl">
            <div className="flex">
              <input
                type="text"
                placeholder="Start typing to search"
                className="flex-grow p-2 text-lg border-2 border-black rounded-l-md focus:outline-none"
              />
              <button className="p-2 text-lg bg-white border-2 border-l-0 border-black rounded-r-md">
                <Search className="w-5 h-5" />
              </button>
            </div>
          </div>
        </div>
      )}

      <div className="flex flex-col lg:flex-row flex-grow">
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6 flex-grow">
          {blogPosts.map((post, index) => (
            <div key={index} className="border-2 border-black rounded-lg overflow-hidden">
              <img
                src={`/placeholder.svg?height=200&width=400&text=Cover+Image+${index + 1}`}
                alt={`Cover image for ${post.title}`}
                className="w-full h-48 object-cover"
              />
              <div className="p-4">
                <div className="flex items-center mb-4">
                  <div className="w-10 h-10 bg-gray-300 rounded-full mr-3"></div>
                  <div>
                    <div className="font-semibold">{post.author}</div>
                    <div className="text-sm text-gray-600">{post.date}</div>
                  </div>
                </div>
                <h2 className="text-xl font-bold mb-2">{post.title}</h2>
                <p className="text-gray-700">{post.content}</p>
              </div>
            </div>
          ))}
        </div>
        <div className="lg:w-64 lg:ml-6 mt-6 lg:mt-0 lg:flex-shrink-0">
          <div className="border-2 border-black rounded-lg p-4 sticky top-6">
            <h3 className="text-xl font-bold mb-2">About GoSuda</h3>
            <p className="text-gray-700 mb-4">
              GoSuda is a leader of innovative solutions for global challenges. Our blog shares insights into our
              technology, culture, and vision for the future.
            </p>
            <h4 className="text-lg font-semibold mb-2">Featured Posts</h4>
            <ul className="space-y-2">
              {featuredPosts.map((post, index) => (
                <li key={index}>
                  <a href={post.link} className="text-blue-600 hover:underline">
                    {post.title}
                  </a>
                </li>
              ))}
            </ul>
          </div>
        </div>
      </div>

      <footer className="mt-8 text-center border-t border-black pt-4">
        <p>&copy; 2024 GoSuda. All rights reserved.</p>
        <div className="mt-2 space-x-4">
          <a href="https://github.com/gosuda" target="_blank" rel="noopener noreferrer" className="text-black">
            GitHub
          </a>
          <a href="https://gosuda.org/editor" target="_blank" rel="noopener noreferrer" className="text-black">
            Editor
          </a>
          <a href="https://gosuda.org" target="_blank" rel="noopener noreferrer" className="text-black">
            Website
          </a>
        </div>
      </footer>
    </div>
  )
}