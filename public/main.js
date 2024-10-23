async function displayAlt() {
  // Language mapping
  const languageMap = {
    en: "English",
    es: "Spanish",
    zh: "Chinese",
    ko: "Korean",
    ja: "Japanese",
    de: "German",
    ru: "Russian",
    fr: "French",
    nl: "Dutch",
    it: "Italian",
    id: "Indonesian",
    pt: "Portuguese",
    sv: "Swedish",
    cs: "Czech",
    sk: "Slovak",
    pl: "Polish",
    ro: "Romanian",
    hu: "Hungarian",
    fi: "Finnish",
    tr: "Turkish"
  };

  const supportedLanguages = [
    "en", "es", "zh", "ko", "ja", "de", "ru", "fr", "nl", "it",
    "id", "pt", "sv", "cs", "sk", "pl", "ro", "hu", "fi", "tr"
  ];
  
  const browserLang = navigator.language.slice(0, 2);
  const pageLang = document.documentElement.lang || 'en';
  
  const alternateLinks = Array.from(document.querySelectorAll('link[rel="alternate"][hreflang]'))
    .reduce((acc, link) => {
      if (link.hreflang !== 'x-default') {
        acc[link.hreflang] = link.href;
      }
      return acc;
    }, {});

  // Only show selector if browser language is supported and different from page language
  if (browserLang !== pageLang && 
      supportedLanguages.includes(browserLang) && 
      alternateLinks[browserLang]) {
    
    const targetUrl = alternateLinks[browserLang];
    const languageName = languageMap[browserLang] || browserLang;
    
    const selector = document.createElement('div');
    selector.style.cssText = `
      margin: 20px auto;
      max-width: 800px;
      background: #fff;
      padding: 15px 20px;
      border-radius: 8px;
      box-shadow: 0 2px 10px rgba(0,0,0,0.1);
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
      ">View in ${languageName}</a>
      <button onclick="this.parentElement.remove()" aria-label="Close Language Selector" style="
        background: none;
        border: none;
        padding: 5px;
        cursor: pointer;
        opacity: 0.5;
        display: flex;
        align-items: center;
        transition: opacity 0.2s;
      " onmouseover="this.style.opacity='1'" onmouseout="this.style.opacity='0.5'">
        <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="18" y1="6" x2="6" y2="18"></line>
          <line x1="6" y1="6" x2="18" y2="18"></line>
        </svg>
      </button>
    `;
    
    const header = document.querySelector('header');
    if (header) {
      header.insertAdjacentElement('afterend', selector);
    } else {
      document.body.insertBefore(selector, document.body.firstChild);
    }
  }
}

async function main() {
  displayAlt();
}

main();
