import react from '@vitejs/plugin-react';
import { defineConfig, type Plugin } from 'vite';

// The policy the built renderer runs under. Nothing is loaded from the
// network — the fonts are bundled, and the daemon is reached through the main
// process, never from here — so the renderer may connect nowhere.
const CSP = [
  "default-src 'self'",
  "script-src 'self'",
  "style-src 'self' 'unsafe-inline'",
  "font-src 'self'",
  "img-src 'self' data:",
  "connect-src 'none'",
  "object-src 'none'",
  "base-uri 'none'",
  "form-action 'none'",
].join('; ');

const CHARSET = '<meta charset="UTF-8" />';

// Injected into the production build only: the dev server needs an inline
// preamble for React refresh and a websocket for HMR, and dev is not what
// ships. Right after the charset, before anything the policy governs.
function contentSecurityPolicy(): Plugin {
  return {
    name: 'dcode:content-security-policy',
    apply: 'build',
    transformIndexHtml(html) {
      if (!html.includes(CHARSET)) throw new Error(`index.html lost its ${CHARSET}; the CSP has nowhere to go`);
      return html.replace(CHARSET, `${CHARSET}\n    <meta http-equiv="Content-Security-Policy" content="${CSP}" />`);
    },
  };
}

export default defineConfig({
  base: './',
  plugins: [react(), contentSecurityPolicy()],
  build: {
    // A font is always its own file: inlined as a data: URI, the policy's
    // font-src 'self' refuses it, and the face falls back without a word.
    assetsInlineLimit: (file) => (/\.(woff2?|ttf|otf)$/.test(file) ? false : undefined),
  },
});
