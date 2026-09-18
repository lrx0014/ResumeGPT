# ResumeGPT homesite

The public marketing/landing page for ResumeGPT — a small, dependency-free Node build script renders one HTML template per locale into static, per-language pages. No frontend framework, no client-side i18n, no runtime JavaScript dependency for content or navigation — just plain HTML/CSS output that's fast and fully crawlable in every language.

```
homesite/
  build.mjs                 Build script: renders src/templates/*.html once per locale into dist/
  package.json               "npm run build" -> dist/, "npm run dev" -> build + serve locally
  src/
    content.json              Shared, non-translated structure (site URL, nav order, icons, image filenames)
    locales/
      en.json, de.json, fr.json, es.json, ja.json, zh-CN.json, zh-TW.json
                               One JSON file per language. en.json is the canonical key shape —
                               every other locale file must mirror it exactly.
    templates/
      index.html               Landing page template (hero, how it works, features, screenshots)
      manual.html               Placeholder "User Manual" page template
    static/
      styles.css               Single stylesheet, CSS custom properties for the ResumeGPT palette
      script.js                 Mobile nav + language-menu toggle + screenshot lightbox, no dependencies
      _headers                  Cloudflare Pages cache-control rules
      assets/
        logo.svg, logo.png       App logo (also used as favicon / og:image)
        screenshots/             Curated, resized screenshots used on the landing page
  dist/                        Build output (git-ignored) — this is what gets deployed
```

## How the build works

`build.mjs` reads `src/content.json` for the list of languages and shared structural data (URLs, screenshot filenames, icons), then for each locale in `src/content.json`'s `languages` array:

1. Loads the matching `src/locales/<code>.json` translation file.
2. Renders `src/templates/index.html` and `src/templates/manual.html`, replacing `{{dotted.key}}` placeholders with values looked up from the locale file (falling back to shared fields from `content.json`, like `{{githubUrl}}`).
3. Generates the repeated blocks (how-it-works steps, feature cards, screenshots, language switcher menu) from `content.json`'s arrays combined with the locale's text.
4. Writes English to `dist/` (site root) and every other language to `dist/<code>/` (e.g. `dist/de/`, `dist/zh-CN/`), so each language gets its own crawlable URL and `hreflang` alternates.
5. Copies `src/static/` (CSS, JS, assets, `_headers`) once into `dist/`, and generates `dist/robots.txt` + `dist/sitemap.xml` listing every page.

Run it with:

```bash
cd homesite
npm run build      # writes dist/
npm run dev         # build, then serve dist/ locally for a quick preview
```

## Deploying to Cloudflare Pages

1. In the Cloudflare dashboard, create a new Pages project connected to this repository.
2. Build settings:
   - **Root directory:** `homesite`
   - **Build command:** `npm run build`
   - **Build output directory:** `dist`
3. Deploy. Every push to the configured branch rebuilds and redeploys automatically.

Before the first deploy, set `siteUrl` in `src/content.json` to the site's real domain — it's used to generate canonical URLs, `hreflang` alternate links, and the sitemap.

## Adding or updating a language

1. Add an entry to the `languages` array in `src/content.json` (`code`, `urlPrefix`, `nativeName`, `shortName`) — this list also drives the language-switcher menu and `apps/web`'s interface-language list, so keep the two in sync where they overlap.
2. Copy `src/locales/en.json` to `src/locales/<code>.json` and translate every value. The key shape must match `en.json` exactly — `build.mjs` throws if a key is missing.
3. `npm run build` and spot-check the new `dist/<code>/` pages.

## Updating screenshots

Screenshots live in `src/static/assets/screenshots/` as standalone copies (not symlinks) so the site has no dependency on `docs/images/` at deploy time. When the app's screenshots change, re-export and resize the relevant ones (max width ~1600px keeps file size reasonable) and drop them in here with the same filenames referenced in `src/content.json`.

## Completing the User Manual

`manual.html` is currently a placeholder ("coming soon" + a link back to the GitHub README), driven by the `manualPage.*` keys in each locale file. Replace the `<section class="placeholder">` block in `src/templates/manual.html` with real content when the manual is written, and update `manualPage.*` in every locale file — the header, footer, and `styles.css` are already shared with `index.html`, so no other changes should be needed.
