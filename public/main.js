
async function main() {
  console.log("gosuda.org/website v0.1");
  
  // Get browser language (first 2 chars for primary language)
  const browserLang = navigator.language.slice(0, 2);
  
  // Get current page language
  const pageLang = document.documentElement.lang || 'en';
  
  // Get alternate links
  const alternateLinks = Array.from(document.querySelectorAll('link[rel="alternate"][hreflang]'))
    .reduce((acc, link) => {
      if (link.hreflang !== 'x-default') {
        acc[link.hreflang] = link.href;
      }
      return acc;
    }, {});

  // Show language selector if:
  // 1. Browser language is different from page language
  // 2. We have an alternate version in browser's language
  if (browserLang !== pageLang && alternateLinks[browserLang]) {
    const targetUrl = alternateLinks[browserLang];
    
    // Create language selector
    const selector = document.createElement('div');
    selector.style.cssText = `
      position: fixed;
      top: 20px;
      right: 20px;
      background: #fff;
      padding: 15px 20px;
      border-radius: 8px;
      box-shadow: 0 2px 10px rgba(0,0,0,0.1);
      z-index: 1000;
      font-size: 14px;
      display: flex;
      align-items: center;
      gap: 10px;
    `;
    
    selector.innerHTML = `
      <span>This content is available in your language</span>
      <a href="${targetUrl}" style="
        background: #4A90E2;
        color: white;
        padding: 5px 15px;
        border-radius: 4px;
        text-decoration: none;
        font-weight: 500;
      ">View in ${browserLang}</a>
      <button onclick="this.parentElement.remove()" style="
        background: none;
        border: none;
        padding: 5px;
        cursor: pointer;
        opacity: 0.5;
      ">×</button>
    `;
    
    document.body.appendChild(selector);
  }
}

main();
