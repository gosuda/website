function isCrawler() {
  const userAgent = navigator.userAgent.toLowerCase();
  const crawlerPattern =
    '(bot|crawler|spider|crawl|agent|fetcher|facebookexternalhit|facebookexternalhit|facebookcatalog|googlebot|baidu|msn|ecosia|instagram|ia_archiver|slack|bing|yeti|yahoo|duckduckgo|linkedin|mediapartners|adsbot)';
  return new RegExp(crawlerPattern, 'i').test(userAgent);
}

// Global alternateLinks variable
let alternateLinks = {};

// Function to get alternateLinks
function getAlternateLinks() {
  return Array.from(
    document.querySelectorAll('link[rel="alternate"][hreflang]')
  ).reduce((acc, link) => {
    if (link.hreflang !== 'x-default') {
      acc[link.hreflang] = link.href;
    }
    return acc;
  }, {});
}

async function displayAlt() {
  if (isCrawler()) return;

  // Language mapping
  const languageMap = {
    en: 'English',
    es: 'Spanish',
    zh: 'Chinese',
    ko: 'Korean',
    ja: 'Japanese',
    de: 'German',
    ru: 'Russian',
    fr: 'French',
    nl: 'Dutch',
    it: 'Italian',
    id: 'Indonesian',
    pt: 'Portuguese',
    sv: 'Swedish',
    cs: 'Czech',
    sk: 'Slovak',
    pl: 'Polish',
    ro: 'Romanian',
    hu: 'Hungarian',
    fi: 'Finnish',
    tr: 'Turkish',
    da: 'Danish',
    no: 'Norwegian',
    bg: 'Bulgarian',
  };

  const supportedLanguages = Object.keys(languageMap);

  const browserLang = navigator.language.slice(0, 2);
  const pageLang = document.documentElement.lang || 'en';

  alternateLinks = getAlternateLinks();

  // Only show selector if browser language is supported and different from page language
  if (
    browserLang !== pageLang &&
    supportedLanguages.includes(browserLang) &&
    alternateLinks[browserLang]
  ) {
    const targetUrl = alternateLinks[browserLang];
    const languageName = languageMap[browserLang] || browserLang;

    const selector = document.createElement('div');
    selector.className =
      'mx-auto my-5 max-w-3xl bg-[var(--surface)] ' +
      'p-4 rounded-lg shadow-md text-sm flex items-center gap-3 ' +
      'border border-[var(--border)]';

    selector.innerHTML = `
    <span class="text-[var(--link)]">This content is available in your language</span>

    <a href="${targetUrl}"
        class="inline-flex items-center rounded-md bg-blue-600 px-4 py-1.5
                font-medium text-white no-underline hover:bg-blue-500
                focus:outline-none focus:ring-2 focus:ring-blue-500
                focus:ring-offset-2 dark:focus:ring-offset-0">
        View in ${languageName}
    </a>

    <button type="button" aria-label="Close Language Selector"
            class="ml-auto inline-flex items-center p-1 opacity-50
                    hover:opacity-100 focus:outline-none focus:ring-2
                    focus:ring-blue-500 focus:ring-offset-2 dark:focus:ring-offset-0">
        <svg class="w-5 h-5" viewBox="0 0 24 24" fill="none"
            stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
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

const FPID = (() => {
  'use strict';

  /**
   * Calculates the SHA-256 hash of a given message.
   * @param {string} message - The string to hash.
   * @returns {Promise<string>} The SHA-256 hash as a hex string.
   */
  const sha256 = async (message) => {
    try {
      const msgUint8 = new TextEncoder().encode(message);
      const hashBuffer = await crypto.subtle.digest('SHA-256', msgUint8);
      const hashArray = Array.from(new Uint8Array(hashBuffer));
      return hashArray.map((b) => b.toString(16).padStart(2, '0')).join('');
    } catch (error) {
      return 'hash_error';
    }
  };

  /**
   * Runs a function multiple times to check for consistent results, a common anti-fingerprinting countermeasure.
   * @param {Function} fn - The fingerprinting function to test.
   * @param {number} [attempts=3] - The number of times to run the check.
   * @returns {Promise<any|string>} The consistent result, or 'Blocked' if inconsistent.
   */
  const runConsistentCheck = async (fn, attempts = 3) => {
    try {
      const firstResult = await fn();
      // 'Blocked' is a definitive state from a sub-check, no need to re-run.
      if (firstResult === 'Blocked') return 'Blocked';
      for (let i = 1; i < attempts; i++) {
        await new Promise((resolve) => setTimeout(resolve, 20));
        if (JSON.stringify(firstResult) !== JSON.stringify(await fn()))
          return 'Blocked';
      }
      return firstResult;
    } catch (error) {
      return 'Error';
    }
  };

  /**
   * A wrapper to run a fingerprinting function and format the output.
   * @param {Function} fn - The fingerprinting function.
   * @returns {Promise<{raw: any, hash: string, status: string}>} An object with the raw value, its hash, and the status.
   */
  const runTest = async (fn) => {
    const result = { raw: 'N/A', hash: 'N/A', status: 'Error' };
    try {
      const raw = await runConsistentCheck(fn);
      if (raw === 'Blocked' || raw === 'Error' || raw === 'NotSupported') {
        result.status = raw;
        result.raw = raw;
      } else {
        result.raw = raw;
        result.hash = await sha256(
          typeof raw === 'string' ? raw : JSON.stringify(raw)
        );
        result.status = 'Success';
      }
    } catch (e) {
      result.raw = e.message;
    }
    return result;
  };

  // --- Private Fingerprinting Modules ---

  const getScreenFingerprint = () =>
    JSON.stringify({
      width: window.screen.width,
      height: window.screen.height,
      devicePixelRatio: window.devicePixelRatio,
      colorDepth: screen.colorDepth,
      pixelDepth: screen.pixelDepth,
      availWidth: screen.availWidth,
      availHeight: screen.availHeight,
      orientation: screen.orientation ? screen.orientation.type : 'N/A',
    });
  const getSensorFingerprint = () => {
    try {
      return JSON.stringify({
        gyroscope: 'activated' in new Gyroscope(),
        accelerometer: 'activated' in new Accelerometer(),
      });
    } catch {
      return 'Error';
    }
  };
  const getFontFingerprint = () => {
    if (!document.fonts?.check) return 'NotSupported';
    const fonts = [
      'Andale Mono',
      'Arial Black',
      'Courier New',
      'Malgun Gothic',
      'Nanum Gothic',
      'Open Sans',
      'Noto Sans',
      'Noto Serif',
      'Adobe Arabic',
      'Acumin',
      'Sloop Script',
      'Cortado',
      'Ubuntu Mono',
      'Big Caslon',
      'Bodoni 72',
      'Yu Gothic',
      'Gulim',
      'Batang',
      'BatangChe',
      'monospace',
      'sans-serif',
    ];
    return fonts.map((font) => document.fonts.check(`10px ${font}`)).join(',');
  };
  const getPluginFingerprint = () => {
    const plugins = navigator.plugins;
    if (!plugins || plugins.length === 0) return 'NoPlugins';
    for (const plugin of plugins) {
      if (plugin.name.toLowerCase().includes('brave')) return 'Blocked';
    }
    return Array.from(plugins)
      .map((p) => `${p.name}|${p.description}|${p.filename}`)
      .join(';');
  };
  const getBrowserApiFingerprint = () =>
    [
      'MathMLElement',
      'PointerEvent',
      'mozInnerScreenX',
      'u2f',
      'WebGL2RenderingContext',
      'SubtleCrypto',
      'Text',
      'Uint8Array',
      'ArrayBuffer',
      'ActiveXObject',
      'Audio',
      'AudioBuffer',
      'AudioBufferSourceNode',
      'Blob',
      'Credential',
      'Gamepad',
      'Geolocation',
      'openDatabase',
      'open',
      'alert',
      'prompt',
      'MouseEvent',
      'RegExp',
      'AuthenticatorResponse',
      'AuthenticatorAttestationResponse',
      'AuthenticatorAssertionResponse',
      'applicationCache',
      'Promise',
      'indexedDB',
      'Cache',
      'CacheStorage',
      'Clipboard',
    ]
      .map((api) => typeof window[api])
      .join(',');
  const getMathFingerprint = () =>
    [
      Math.PI,
      Math.E,
      Math.LN2,
      Math.LN10,
      Math.SQRT1_2,
      Math.SQRT2,
      Math.sin(10),
      Math.sinh(10),
      Math.cos(10),
      Math.cosh(10),
    ].join(',');
  const getWebGLFingerprint = () => {
    const c = document.createElement('canvas'),
      gl = c.getContext('webgl') || c.getContext('experimental-webgl');
    if (!gl) return 'NotSupported';
    try {
      const d = gl.getExtension('WEBGL_debug_renderer_info');
      return JSON.stringify({
        vendor: d ? gl.getParameter(d.UNMASKED_VENDOR_WEBGL) : 'N/A',
        renderer: d ? gl.getParameter(d.UNMASKED_RENDERER_WEBGL) : 'N/A',
        extensions: gl.getSupportedExtensions()?.join(','),
      });
    } catch (e) {
      return 'Error';
    } finally {
      c.remove();
    }
  };
  const getJsonOrderFingerprint = () =>
    JSON.stringify({ z: 1, y: 'test', x: true, a: [1, 2, 3], b: null });
  const getBatteryFingerprint = () =>
    (typeof navigator.getBattery === 'function').toString();
  const getHardwareApisFingerprint = () =>
    JSON.stringify({
      hid: 'hid' in navigator,
      usb: 'usb' in navigator,
      serial: 'serial' in navigator,
    });
  const getIntlFingerprint = () => {
    try {
      const o = Intl.DateTimeFormat().resolvedOptions();
      return JSON.stringify({
        locale: o.locale,
        timeZone: o.timeZone,
        numberingSystem: o.numberingSystem,
      });
    } catch (e) {
      return 'Error';
    }
  };

  const validateCanvasPixels = () => {
    try {
      const canvas = document.createElement('canvas');
      canvas.width = 100;
      canvas.height = 50;
      const ctx = canvas.getContext('2d');
      if (!ctx) return false;
      ctx.fillStyle = 'rgb(255, 255, 255)';
      ctx.fillRect(0, 0, canvas.width, canvas.height);
      ctx.fillStyle = 'rgb(0, 0, 0)';
      ctx.fillRect(10, 10, 80, 30);
      const pointsToCheck = {
        white: [
          [5, 5],
          [95, 5],
          [5, 45],
          [95, 45],
        ],
        black: [
          [15, 15],
          [85, 15],
          [15, 35],
          [85, 35],
        ],
      };
      for (const point of pointsToCheck.white) {
        const p = ctx.getImageData(point[0], point[1], 1, 1).data;
        if (p[0] !== 255 || p[1] !== 255 || p[2] !== 255 || p[3] !== 255)
          return false;
      }
      for (const point of pointsToCheck.black) {
        const p = ctx.getImageData(point[0], point[1], 1, 1).data;
        if (p[0] !== 0 || p[1] !== 0 || p[2] !== 0 || p[3] !== 255)
          return false;
      }
      return true;
    } catch (e) {
      return false;
    }
  };

  const getCanvasFingerprint = () => {
    if (!validateCanvasPixels()) return 'Blocked';
    const c = document.createElement('canvas');
    c.width = 300;
    c.height = 200;
    const x = c.getContext('2d');
    if (!x) return 'NotSupported';
    try {
      const g = x.createLinearGradient(0, 0, 0, 150);
      g.addColorStop(0, 'black');
      g.addColorStop(1, 'gray');
      x.fillStyle = g;
      x.fillRect(0, 0, 300, 200);
      x.fillStyle = 'white';
      x.shadowBlur = 10;
      x.shadowColor = 'yellow';
      x.font = "16px 'Noto Sans'";
      x.fillText(
        'Hello World! 12345 &^%$#?\\/�🍳🍔🍟🍤😫🙄😑😐🤗😀ΑΕγξΕηθτξΞ',
        10,
        20
      );
      x.beginPath();
      x.arc(75, 75, 50, 0, Math.PI * 2, true);
      x.moveTo(110, 75);
      x.arc(75, 75, 35, 0, Math.PI, false);
      x.moveTo(65, 65);
      x.arc(60, 65, 5, 0, Math.PI * 2, true);
      x.moveTo(95, 65);
      x.arc(90, 65, 5, 0, Math.PI * 2, true);
      x.strokeStyle = 'rgba(0, 255, 0, 0.7)';
      x.stroke();
      return c.toDataURL('image/png');
    } catch (e) {
      return 'Error';
    } finally {
      c.remove();
    }
  };

  const validateAudioContext = (buffer) => {
    if (!buffer) return false;
    let firstValue = buffer[0];
    for (let i = 1; i < buffer.length; i++) {
      if (buffer[i] !== firstValue) return true;
    }
    return false;
  };

  const getAudioContextFingerprint = () =>
    new Promise((r) => {
      try {
        if (!!JSON.stringify(navigator.userAgentData).match(/Brave/))
          return r('Blocked');
        const a = new (window.OfflineAudioContext ||
          window.webkitOfflineAudioContext)(1, 44100, 44100);
        if (!a) return r('NotSupported');
        const o = a.createOscillator();
        o.type = 'triangle';
        o.frequency.setValueAtTime(10000, a.currentTime);
        const c = a.createDynamicsCompressor();
        c.threshold.setValueAtTime(-50, a.currentTime);
        c.knee.setValueAtTime(40, a.currentTime);
        c.ratio.setValueAtTime(12, a.currentTime);
        c.attack.setValueAtTime(0, a.currentTime);
        c.release.setValueAtTime(0.25, a.currentTime);
        o.connect(c);
        c.connect(a.destination);
        o.start(0);
        a.startRendering();
        a.oncomplete = (e) => {
          const buffer = e.renderedBuffer.getChannelData(0);
          if (!validateAudioContext(buffer)) return r('Blocked');
          r(
            buffer
              .slice(4500, 5000)
              .reduce((s, v) => s + Math.abs(v), 0)
              .toString()
          );
        };
      } catch (e) {
        r('Error');
      }
    });

  // --- Public Method ---

  /**
   * Generates a comprehensive browser fingerprint.
   * @returns {Promise<Object>} A promise that resolves to an object containing all fingerprinting results and a final hash.
   */
  const generate = async () => {
    const results = {
      canvas: await runTest(getCanvasFingerprint),
      audio: await runTest(getAudioContextFingerprint),
      webgl: await runTest(getWebGLFingerprint),
      fonts: await runTest(getFontFingerprint),
      screen: await runTest(getScreenFingerprint),
      intl: await runTest(getIntlFingerprint),
      sensors: await runTest(getSensorFingerprint),
      plugins: await runTest(getPluginFingerprint),
      browserApis: await runTest(getBrowserApiFingerprint),
      hardwareApis: await runTest(getHardwareApisFingerprint),
      battery: await runTest(getBatteryFingerprint),
      math: await runTest(getMathFingerprint),
      jsonOrder: await runTest(getJsonOrderFingerprint),
    };
    const finalHashes = Object.values(results)
      .map((v) => v.hash)
      .sort()
      .join('');
    results.finalHash = await sha256(finalHashes);

    return results;
  };

  return {
    generate: generate,
  };
})();

//@@START_CONFIG@@
const TELEMETRY_FP_VERSION = 1;
const TELEMETRY_BASEURL = 'https://telemetry.gosuda.org';
const CLIENT_VERSION = '20250810-V1BETA1';
//@@END_CONFIG@@

/**
 * Checks if the client is already registered by verifying credentials with the telemetry server.
 * Corresponds to POST /client/status API endpoint.
 * @returns {Promise<boolean>} - True if the client is registered and valid, false otherwise.
 */
async function checkClientStatus() {
  let clientID = localStorage.getItem('telemetry_client_id');
  let clientToken = localStorage.getItem('telemetry_client_token');

  if (!clientID || !clientToken) {
    return false;
  }

  try {
    const resp = await fetch(TELEMETRY_BASEURL + '/client/status', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        id: clientID,
        token: clientToken,
      }),
    });

    return resp.status === 200;
  } catch (error) {
    console.error('Error checking client status:', error);
    return false;
  }
}

/**
 * Registers a new client with the telemetry server.
 * Corresponds to POST /client/register API endpoint.
 * Stores the received client ID and token in local storage.
 * @returns {Promise<Object>} - The client identity (id and token).
 * @throws {Error} If registration fails.
 */
async function registerClient() {
  const resp = await fetch(TELEMETRY_BASEURL + '/client/register', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
  });
  if (resp.status !== 201) {
    throw new Error(`Failed to register client: Status ${resp.status}`);
  }

  const clientIdentity = await resp.json();
  localStorage.setItem('telemetry_client_id', clientIdentity.id);
  localStorage.setItem('telemetry_client_token', clientIdentity.token);

  return clientIdentity;
}

/**
 * Registers the browser fingerprint with the telemetry server.
 * Corresponds to POST /client/checkin API endpoint.
 * Includes client credentials, version details, and user agent information.
 * @param {string} fingerprint - The generated browser fingerprint hash.
 * @throws {Error} If fingerprint registration fails.
 */
async function registerFingerprint(fingerprint) {
  let clientID = localStorage.getItem('telemetry_client_id');
  let clientToken = localStorage.getItem('telemetry_client_token');

  if (!clientID || !clientToken) {
    throw new Error('Client not registered. Cannot register fingerprint.');
  }

  const resp = await fetch(TELEMETRY_BASEURL + '/client/checkin', {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      client_id: clientID,
      client_token: clientToken,
      version: CLIENT_VERSION,
      fpv: TELEMETRY_FP_VERSION,
      fp: fingerprint,
      ua: navigator.userAgent,
      uad: JSON.stringify(navigator.userAgentData),
    }),
  });
  if (resp.status !== 200) {
    throw new Error(`Failed to register fingerprint: Status ${resp.status}`);
  }
}

/**
 * Records a page view for the current URL
 * @param {string} url - The URL to record a view for (defaults to current page URL)
 * @returns {Promise<boolean>} - Returns true if view was recorded successfully
 */
async function recordView(url = window.location.href) {
  let clientID = localStorage.getItem('telemetry_client_id');
  let clientToken = localStorage.getItem('telemetry_client_token');

  if (!clientID || !clientToken) {
    console.warn('Client not registered. Cannot record view.');
    return false;
  }

  try {
    const resp = await fetch(TELEMETRY_BASEURL + '/client/view', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        client_id: clientID,
        client_token: clientToken,
        url: url,
      }),
    });

    if (resp.status === 200) {
      console.log('View recorded successfully for:', url);
      return true;
    } else {
      console.error('Failed to record view. Status:', resp.status);
      return false;
    }
  } catch (error) {
    console.error('Error recording view:', error);
    return false;
  }
}

/**
 * Gets the view count for a specific URL
 * @param {string} url - The URL to get view count for (defaults to current page URL)
 * @returns {Promise<Object|null>} - Returns view count data or null if failed
 */
async function getViewCount(url = window.location.href) {
  try {
    const resp = await fetch(
      TELEMETRY_BASEURL +
        '/view/count?' +
        new URLSearchParams({
          url: url,
        }),
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      }
    );

    if (resp.status === 200) {
      const data = await resp.json();
      console.log('View count for', url + ':', data.count);
      return data;
    } else if (resp.status === 404) {
      console.log('URL not found, view count is 0 for:', url);
      return { url: url, count: 0 };
    } else {
      console.error('Failed to get view count. Status:', resp.status);
      return null;
    }
  } catch (error) {
    console.error('Error getting view count:', error);
    return null;
  }
}

/**
 * Records a view and optionally returns the updated view count
 * @param {string} url - The URL to record a view for (defaults to current page URL)
 * @param {boolean} returnCount - Whether to return the updated view count
 * @returns {Promise<Object|boolean>} - Returns view count data if returnCount is true, otherwise boolean success
 */
async function recordViewAndGetCount(
  url = window.location.href,
  returnCount = true
) {
  const viewRecorded = await recordView(url);

  if (!viewRecorded) {
    return returnCount ? null : false;
  }

  if (returnCount) {
    // Wait a bit for the database to be updated
    await new Promise((resolve) => setTimeout(resolve, 100));
    return await getViewCount(url);
  }

  return true;
}

/**
 * Records a "like" for the current URL
 * @param {string} url - The URL to record a like for (defaults to current page URL)
 * @returns {Promise<boolean>} - Returns true if like was recorded successfully
 */
async function recordLike(url = window.location.href) {
  let clientID = localStorage.getItem('telemetry_client_id');
  let clientToken = localStorage.getItem('telemetry_client_token');

  if (!clientID || !clientToken) {
    console.warn('Client not registered. Cannot record like.');
    return false;
  }

  try {
    const resp = await fetch(TELEMETRY_BASEURL + '/client/like', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({
        client_id: clientID,
        client_token: clientToken,
        url: url,
      }),
    });

    if (resp.status === 200) {
      console.log('Like recorded successfully for:', url);
      return true;
    } else {
      console.error('Failed to record like. Status:', resp.status);
      return false;
    }
  } catch (error) {
    console.error('Error recording like:', error);
    return false;
  }
}

/**
 * Gets the like count for a specific URL
 * @param {string} url - The URL to get like count for (defaults to current page URL)
 * @returns {Promise<Object|null>} - Returns like count data or null if failed
 */
async function getLikeCount(url = window.location.href) {
  try {
    const resp = await fetch(
      TELEMETRY_BASEURL +
        '/like/count?' +
        new URLSearchParams({
          url: url,
        }),
      {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
        },
      }
    );

    if (resp.status === 200) {
      const data = await resp.json();
      console.log('Like count for', url + ':', data.count);
      return data;
    } else if (resp.status === 404) {
      console.log('URL not found, like count is 0 for:', url);
      return { url: url, count: 0 };
    } else {
      console.error('Failed to get like count. Status:', resp.status);
      return null;
    }
  } catch (error) {
    console.error('Error getting like count:', error);
    return null;
  }
}

/**
 * Main telemetry function to ensure client registration, fingerprint check-in, and page view recording.
 * Handles initial client registration if needed and updates fingerprint if changed.
 */
async function telemetry() {
  let clientFingerprint = localStorage.getItem('telemetry_client_fingerprint');

  // Ensure the client is registered. If not, register it.
  if (!(await checkClientStatus())) {
    try {
      await registerClient();
      // Re-check status after registration to confirm
      const ok = await checkClientStatus();
      if (!ok) {
        throw new Error('Client registration confirmed failed after attempt.');
      }
    } catch (error) {
      console.error('Initial client registration process failed:', error);
      // Depending on desired behavior, might stop telemetry here or try again later
      return;
    }
  }

  // Generate current fingerprint and check if it has changed since last visit.
  const fp = await FPID.generate();
  console.log('Generated Fingerprint Hash:', fp.finalHash);

  if (fp.finalHash !== clientFingerprint) {
    try {
      await registerFingerprint(fp.finalHash);
      localStorage.setItem('telemetry_client_fingerprint', fp.finalHash);
      console.log('New fingerprint registered and stored.');
    } catch (error) {
      console.error('Failed to register new fingerprint:', error);
    }
  } else {
    console.log('Fingerprint is unchanged.');
  }

  // Record views only for pages that have view-count placeholders (data-view-count).
  try {
    const viewEls = document.querySelectorAll('[data-view-count]');
    if (viewEls.length > 0) {
      // Use element's data-url only (do NOT fall back to canonical or current location).
      const urls = Array.from(viewEls)
        .map((el) => el.getAttribute('data-url'))
        .filter(Boolean);
      // Deduplicate URLs and record views for each
      const uniqueUrls = [...new Set(urls)];
      await Promise.all(uniqueUrls.map((u) => recordView(u)));
    } else {
      // No view placeholders on this page; skip recording.
    }
  } catch (error) {
    console.error('Failed to record page view(s):', error);
  }
}

/**
 * Initialize telemetry and automatically record page views
 */
async function initTelemetry() {
  const runHydrations = () => {
    try {
      hydrateCounts();
    } catch (e) {
      console.error('hydrateCounts failed:', e);
    }
    try {
      hydrateSummaryCounts();
    } catch (e) {
      console.error('hydrateSummaryCounts failed:', e);
    }
  };

  // pre-hydrate
  runHydrations();

  // run telemetry
  try {
    await telemetry();
  } catch (error) {
    console.error('Telemetry initialization failed:', error);
  }

  // post-hydrate
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', runHydrations, {
      once: true,
    });
  } else {
    runHydrations();
  }
}

// Auto-initialize when the script loads
initTelemetry();

// Hydrate view and like placeholders inserted server-side.
async function hydrateCounts() {
  if (isCrawler()) return;

  // Views
  const viewEls = document.querySelectorAll('[data-view-count]');
  for (const el of viewEls) {
    const url = el.getAttribute('data-url');
    if (!url) continue;
    try {
      const data = await getViewCount(url);
      if (data && typeof data.count !== 'undefined') {
        el.textContent = `views ${data.count}`;
      } else {
        el.textContent = 'views 0';
      }
    } catch (e) {
      el.textContent = 'views 0';
    }
  }

  // Likes
  const likeButtons = document.querySelectorAll('[data-like-button]');
  for (const btn of likeButtons) {
    const url = btn.getAttribute('data-url');
    if (!url) continue;
    const span = btn.querySelector('[data-like-count]');
    try {
      const data = await getLikeCount(url);
      const count = data && typeof data.count !== 'undefined' ? data.count : 0;
      if (span) span.textContent = `likes ${count}`;
    } catch (e) {
      if (span) span.textContent = 'likes 0';
    }

    // Attach click handler to record a like and update UI optimistically
    btn.addEventListener('click', async (ev) => {
      ev.preventDefault();
      if (!span) return;
      const numeric =
        parseInt((span.textContent || '').replace(/\D/g, ''), 10) || 0;
      // optimistic update
      span.textContent = `likes ${numeric + 1}`;
      try {
        const ok = await recordLike(url);
        if (!ok) {
          // revert on failure
          span.textContent = `likes ${numeric}`;
          return;
        }
        // confirm with server
        const fresh = await getLikeCount(url);
        if (fresh && typeof fresh.count !== 'undefined') {
          span.textContent = `likes ${fresh.count}`;
        }
      } catch (e) {
        span.textContent = `likes ${numeric}`;
      }
    });
  }
}

/**
 * Hydrate summary counts for index/list pages using bulk lookup. Uses data-summary-url,
 * data-summary-view-count, and data-summary-like-count. Completely separate from per-page
 * hydration logic.
 */
async function hydrateSummaryCounts() {
  if (isCrawler()) return;
  // Collect URLs from summary placeholders
  const summaryEls = Array.from(
    document.querySelectorAll('[data-summary-url]')
  );
  if (summaryEls.length === 0) return;
  const urls = summaryEls
    .map((el) => el.getAttribute('data-summary-url'))
    .filter(Boolean);
  const uniqueUrls = [...new Set(urls)];
  try {
    const bulk = await getBulkCounts(uniqueUrls);
    if (!bulk || !bulk.map) return;
    const map = bulk.map;
    // Update view placeholders
    const viewEls = document.querySelectorAll('[data-summary-view-count]');
    for (const el of viewEls) {
      const url = el.getAttribute('data-summary-url');
      if (!url) continue;
      const entry = map[url];
      const count =
        entry && typeof entry.view_count !== 'undefined' ? entry.view_count : 0;
      el.textContent = `views ${count}`;
    }
    // Update like placeholders
    const likeEls = document.querySelectorAll('[data-summary-like-count]');
    for (const el of likeEls) {
      const url = el.getAttribute('data-summary-url');
      if (!url) continue;
      const entry = map[url];
      const count =
        entry && typeof entry.like_count !== 'undefined' ? entry.like_count : 0;
      el.textContent = `likes ${count}`;
    }
  } catch (e) {
    console.error('hydrateSummaryCounts failed:', e);
    for (const el of document.querySelectorAll('[data-summary-view-count]')) {
      el.textContent = 'views 0';
    }
    for (const el of document.querySelectorAll('[data-summary-like-count]')) {
      el.textContent = 'likes 0';
    }
  }
}

/**
 * Performs a bulk counts lookup for multiple URLs.
 * POST /counts/bulk expects JSON body: { "urls": ["https://...", ...] }
 * Returns an object with the raw server response and a convenience map keyed by normalized URL:
 *   { raw: { results: [...] }, map: { "https://...": { view_count, like_count }, ... } }
 * or null on error.
 * @param {string[]} urls
 * @returns {Promise<Object|null>}
 */
async function getBulkCounts(urls = []) {
  if (!Array.isArray(urls) || urls.length === 0) {
    console.warn('getBulkCounts: expected a non-empty array of urls');
    return { raw: { results: [] }, map: {} };
  }

  try {
    const resp = await fetch(TELEMETRY_BASEURL + '/counts/bulk', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ urls: urls }),
    });

    if (resp.status === 200) {
      const data = await resp.json();
      const map = {};
      if (Array.isArray(data.results)) {
        for (const e of data.results) {
          map[e.url] = { view_count: e.view_count, like_count: e.like_count };
        }
      }
      return { raw: data, map: map };
    } else {
      console.error('Failed to get bulk counts. Status:', resp.status);
      return null;
    }
  } catch (error) {
    console.error('Error getting bulk counts:', error);
    return null;
  }
}

// Make functions available globally for manual use
window.recordView = recordView;
window.getViewCount = getViewCount;
window.recordViewAndGetCount = recordViewAndGetCount;
window.recordLike = recordLike;
window.getLikeCount = getLikeCount;
window.getBulkCounts = getBulkCounts;
window.hydrateSummaryCounts = hydrateSummaryCounts;
const THEME_KEY = 'pref_theme'; // 'light' | 'dark' | 'system'
function themePrefersDark() {
  return (
    window.matchMedia &&
    window.matchMedia('(prefers-color-scheme: dark)').matches
  );
}
function themeGet() {
  try {
    return localStorage.getItem(THEME_KEY) || 'system';
  } catch {
    return 'system';
  }
}
function themeApply(theme) {
  const dark = theme === 'dark' || (theme === 'system' && themePrefersDark());
  const root = document.documentElement;
  root.classList.toggle('dark', dark);
  root.setAttribute('data-theme', dark ? 'dark' : 'light');
  document.querySelectorAll('[data-theme-toggle]').forEach((btn) => {
    btn.setAttribute('aria-pressed', String(dark));
    const icon = btn.querySelector('[data-theme-icon]');
    if (icon) icon.textContent = dark ? '🌙' : '☀️';
  });
}
function themeSet(theme) {
  try {
    localStorage.setItem(THEME_KEY, theme);
  } catch {}
  themeApply(theme);
}
function themeToggle() {
  const cur = themeGet();
  if (cur === 'system') themeSet(themePrefersDark() ? 'light' : 'dark');
  else themeSet(cur === 'dark' ? 'light' : 'dark');
}

/* React to OS changes when user is on "system" */
if (window.matchMedia) {
  const mq = window.matchMedia('(prefers-color-scheme: dark)');
  mq.addEventListener?.('change', () => {
    if (themeGet() === 'system') themeApply('system');
  });
}

/* Wire up the mounted button and apply current theme */
function initTheme() {
  themeApply(themeGet());
  const btn = document.querySelector('[data-theme-toggle]');
  if (btn) {
    btn.addEventListener('click', (e) => {
      e.preventDefault();
      themeToggle();
    });
  }
}
if (document.readyState === 'loading') {
  document.addEventListener('DOMContentLoaded', initTheme, { once: true });
} else {
  initTheme();
}

const svg = {
  ko: `/assets/images/flag/kr.svg`,
  en: `/assets/images/flag/gb.svg`,
  ja: `/assets/images/flag/jp.svg`,
  es: `/assets/images/flag/es.svg`,
  zh: `/assets/images/flag/cn.svg`,
  de: `/assets/images/flag/de.svg`,
  ru: `/assets/images/flag/ru.svg`,
  fr: `/assets/images/flag/fr.svg`,
  nl: `/assets/images/flag/nl.svg`,
  it: `/assets/images/flag/it.svg`,
  id: `/assets/images/flag/id.svg`,
  pt: `/assets/images/flag/pt.svg`,
  sv: `/assets/images/flag/sv.svg`,
  cs: `/assets/images/flag/cz.svg`,
  sk: `/assets/images/flag/sk.svg`,
  pl: `/assets/images/flag/pl.svg`,
  ro: `/assets/images/flag/ro.svg`,
  hu: `/assets/images/flag/hu.svg`,
  fi: `/assets/images/flag/fi.svg`,
  tr: `/assets/images/flag/tr.svg`,
  da: `/assets/images/flag/dk.svg`,
  no: `/assets/images/flag/no.svg`,
  bg: `/assets/images/flag/bg.svg`,
};

// Language mapping
const languageMap = {
  ko: 'Korean',
  en: 'English',
  es: 'Spanish',
  zh: 'Chinese',
  ja: 'Japanese',
  de: 'German',
  ru: 'Russian',
  fr: 'French',
  nl: 'Dutch',
  it: 'Italian',
  id: 'Indonesian',
  pt: 'Portuguese',
  sv: 'Swedish',
  cs: 'Czech',
  sk: 'Slovak',
  pl: 'Polish',
  ro: 'Romanian',
  hu: 'Hungarian',
  fi: 'Finnish',
  tr: 'Turkish',
  da: 'Danish',
  no: 'Norwegian',
  bg: 'Bulgarian',
};

document.addEventListener('DOMContentLoaded', function () {
  const dropdownButton = document.querySelector('.dropdown-button');
  const dropdownContent = document.querySelector('.dropdown-content');

  const pageLang = document.documentElement.lang || 'en';
  const currentLang = pageLang || 'en';
  dropdownButton.innerHTML = `<img src="${svg[currentLang]}" alt="${languageMap[currentLang]} flag" width="32" height="32"> ${languageMap[currentLang]} ▲`;

  // Get alternateLinks on page load
  alternateLinks = getAlternateLinks();

  // Create dropdown items based on alternateLinks
  Object.keys(alternateLinks).forEach((key, index) => {
    // Only create dropdown item if we have the language mapping and svg
    if (languageMap[key] && svg[key]) {
      const liBtn = document.createElement('button');
      liBtn.className = 'dropdown-item';
      liBtn.style.setProperty('--order', index);
      liBtn.innerHTML = `<img id="${key}" src="${svg[key]}" alt="${languageMap[key]} flag" width="32" height="32"> ${languageMap[key]}`;

      liBtn.addEventListener('click', function () {
        dropdownButton.innerHTML = `<img src="${svg[key]}" alt="${languageMap[key]} flag" width="32" height="32"> ${languageMap[key]}`;
        window.location.href = alternateLinks[key];
        dropdownContent.classList.remove('show');
      });

      dropdownContent.appendChild(liBtn);
    }
  });

  dropdownButton.addEventListener('click', function () {
    dropdownContent.classList.toggle('show');

    // 드롭다운 상태에 따라 화살표 변경
    if (dropdownContent.classList.contains('show')) {
      // 열린 상태: 아래쪽 화살표
      const currentContent = dropdownButton.innerHTML.replace(/[▲▼]/g, '');
      dropdownButton.innerHTML = currentContent + ' ▼';
    } else {
      // 닫힌 상태: 위쪽 화살표
      const currentContent = dropdownButton.innerHTML.replace(/[▲▼]/g, '');
      dropdownButton.innerHTML = currentContent + ' ▲';
    }
  });

  window.addEventListener('click', function (event) {
    if (!event.target.matches('.dropdown-button')) {
      if (dropdownContent.classList.contains('show')) {
        dropdownContent.classList.remove('show');
        // 닫힌 상태로 화살표 변경
        const currentContent = dropdownButton.innerHTML.replace(/[▲▼]/g, '');
        dropdownButton.innerHTML = currentContent + ' ▲';
      }
    }
  });
});
