# Client — React + Vite + TypeScript + Tailwind CSS v3

This is the Rarefactor frontend, a single‑page React app built with Vite and styled using Tailwind CSS v3. It talks to the FastAPI backend over HTTP for autocomplete and search.

## Stack

- React 19 + TypeScript
- Vite 7
- Tailwind CSS v3 (with PostCSS + Autoprefixer)

Note: We previously experimented with Mantine, but the UI has been migrated to Tailwind. Mantine is no longer used by the app at runtime. Some legacy files may still reference Mantine and can be cleaned up in a future pass.

## Project layout (client)

- `src/App.tsx` — entry component rendering the unified search interface
- `src/components/SearchInterface.tsx` — combined search input + suggestions + results
- `src/hooks/useAutocomplete.ts` — fetches suggestions from the backend
- `index.css` — Tailwind directives and minimal base styles
- `tailwind.config.js` — Tailwind v3 configuration
- `postcss.config.js` — standard PostCSS setup

## Backend contract

The UI expects the backend (FastAPI) to expose:
- `GET /autocomplete?q=term&limit=10` → `{ suggestions: string[] }`
- `GET /search?q=query` → `{ results: Array<{ title?: string; url?: string; snippet?: string }> }`

See `../backend/README.md` for backend details.

## Development

Prerequisites: Node 18+ (Node 20 recommended). You may see a Vite warning if your Node is below the suggested version; it does not block development in this demo.

1. Install dependencies:
   ```bash
   npm install
   ```
2. Start the dev server:
   ```bash
   npm run dev
   ```
3. Open http://localhost:5173

Make sure the backend is running at http://localhost:8000.

## Build & preview

```bash
npm run build
npm run preview
```

## Tailwind setup

Tailwind v3 is configured with a standard PostCSS stack.

- `tailwind.config.js`
  ```js
  /** @type {import('tailwindcss').Config} */
  module.exports = {
    content: ['./index.html', './src/**/*.{js,ts,jsx,tsx}'],
    theme: { extend: {} },
    plugins: [],
  };
  ```
- `postcss.config.js`
  ```js
  export default {
    plugins: { tailwindcss: {}, autoprefixer: {} },
  }
  ```
- `src/index.css`
  ```css
  @tailwind base;
  @tailwind components;
  @tailwind utilities;
  @layer base { html, body, #root { height: 100%; } body { @apply bg-white text-gray-900 antialiased; } }
  ```

## Scripts

- `npm run dev` — start Vite dev server
- `npm run build` — type‑check and build for production
- `npm run preview` — preview the production build

## Known notes

- If you see a Node.js engine warning from Vite (requesting a newer 20.x), it’s informational — builds and dev server still work for this project.
