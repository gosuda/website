/** @type {import('tailwindcss').Config} */

import typography from '@tailwindcss/typography';
export default {
  content: ["./{view,public,templates}/**/*.{html,js,templ}"],
  theme: {
    extend: {},
    fontFamily: {
      sans: [
        "'IBM Plex Sans KR'", "Roboto", "system-ui", "-apple-system", "BlinkMacSystemFont", "'Segoe UI'", "Oxygen", "Ubuntu", "Cantarell", "'Open Sans'", "'Helvetica Neue'", "sans-serif"
      ]
    },
    fontWeight: {
      "normal": 300,
      "bold": 500
    },
  },
  plugins: [
    typography,
  ],
}

