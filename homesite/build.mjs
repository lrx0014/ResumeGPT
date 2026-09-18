// Zero-dependency static site generator for the ResumeGPT marketing homesite.
// Renders src/templates/*.html once per locale in src/locales/*.json into dist/.
import { readFileSync, writeFileSync, mkdirSync, cpSync, rmSync, readdirSync } from "node:fs";
import { join, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const ROOT = dirname(fileURLToPath(import.meta.url));
const SRC = join(ROOT, "src");
const DIST = join(ROOT, "dist");

const content = JSON.parse(readFileSync(join(SRC, "content.json"), "utf8"));

function loadLocale(code) {
  return JSON.parse(readFileSync(join(SRC, "locales", `${code}.json`), "utf8"));
}

const locales = content.languages.map((lang) => ({
  ...lang,
  t: loadLocale(lang.code),
}));

function get(obj, path) {
  const value = path.split(".").reduce((acc, key) => (acc == null ? undefined : acc[key]), obj);
  if (value === undefined) {
    throw new Error(`Missing translation key "${path}"`);
  }
  return value;
}

function renderTemplate(template, view) {
  return template.replace(/\{\{([\w.]+)\}\}/g, (match, key) => {
    if (key in view) return view[key];
    return String(get(view.__t, key));
  });
}

function buildSteps(t) {
  return content.steps
    .map((step, index) => {
      const copy = get(t, `howItWorks.steps.${step.id}`);
      const arrow = index < content.steps.length - 1 ? `<span class="step-arrow">→</span>` : "";
      return `<div class="step">
          <div class="step-number">${index + 1}</div>
          <h3>${copy.title}</h3>
          <p>${copy.desc}</p>
          ${arrow}
        </div>`;
    })
    .join("\n        ");
}

function buildFeatures(t) {
  return content.features
    .map((feature) => {
      const copy = get(t, `features.items.${feature.id}`);
      return `<div class="feature-card">
          <div class="feature-icon">${feature.icon}</div>
          <h3>${copy.title}</h3>
          <p>${copy.desc}</p>
        </div>`;
    })
    .join("\n        ");
}

function buildShots(t, assetPrefix) {
  return content.screenshots
    .map((shot) => {
      const copy = get(t, `screenshots.items.${shot.id}`);
      const src = `${assetPrefix}assets/screenshots/${shot.file}`;
      const caption = `${copy.title} — ${copy.desc}`;
      return `<button class="shot" type="button" data-full="${src}" data-caption="${caption}">
          <img src="${src}" alt="${shot.alt}" loading="lazy" />
          <span class="shot-caption"><strong>${copy.title}</strong>${copy.desc}</span>
        </button>`;
    })
    .join("\n        ");
}

function buildLangMenu(currentCode, pageFile) {
  return locales
    .map((lang) => {
      const active = lang.code === currentCode ? " active" : "";
      return `<a href="${lang.urlPrefix}${pageFile}" class="${active.trim()}" lang="${lang.code}">${lang.nativeName}</a>`;
    })
    .join("\n            ");
}

function buildAlternateLinks(pageFile) {
  const tags = locales
    .map((lang) => `  <link rel="alternate" hreflang="${lang.code}" href="${content.siteUrl}${lang.urlPrefix}${pageFile}" />`)
    .join("\n");
  const xDefault = `  <link rel="alternate" hreflang="x-default" href="${content.siteUrl}/${pageFile}" />`;
  return `${tags}\n${xDefault}`;
}

function renderPage(templateName, pageFile, lang) {
  const template = readFileSync(join(SRC, "templates", templateName), "utf8");
  const assetPrefix = lang.urlPrefix === "/" ? "" : "../";
  const isManual = pageFile === "manual.html";

  const view = {
    __t: lang.t,
    htmlLang: lang.t.htmlLang,
    ...content,
    ASSET_PREFIX: assetPrefix,
    CANONICAL: `${content.siteUrl}${lang.urlPrefix}${pageFile}`,
    ALTERNATE_LINKS: buildAlternateLinks(pageFile),
    STEPS: buildSteps(lang.t),
    FEATURES: buildFeatures(lang.t),
    SHOTS: buildShots(lang.t, assetPrefix),
    LANG_MENU: buildLangMenu(lang.code, pageFile),
    LANG_LABEL: lang.shortName,
    HREF_HOME: "./",
    HREF_MANUAL: "manual.html",
    HREF_HOW: isManual ? "index.html#how-it-works" : "#how-it-works",
    HREF_FEATURES: isManual ? "index.html#features" : "#features",
    HREF_SHOTS: isManual ? "index.html#screenshots" : "#screenshots",
  };

  return renderTemplate(template, view);
}

function main() {
  rmSync(DIST, { recursive: true, force: true });
  mkdirSync(DIST, { recursive: true });

  cpSync(join(SRC, "static", "assets"), join(DIST, "assets"), { recursive: true });
  cpSync(join(SRC, "static", "styles.css"), join(DIST, "styles.css"));
  cpSync(join(SRC, "static", "script.js"), join(DIST, "script.js"));
  cpSync(join(SRC, "static", "_headers"), join(DIST, "_headers"));

  const sitemapUrls = [];

  for (const lang of locales) {
    const outDir = lang.urlPrefix === "/" ? DIST : join(DIST, lang.urlPrefix.slice(1, -1));
    mkdirSync(outDir, { recursive: true });

    for (const [templateName, pageFile] of [
      ["index.html", "index.html"],
      ["manual.html", "manual.html"],
    ]) {
      const html = renderPage(templateName, pageFile, lang);
      writeFileSync(join(outDir, pageFile), html);
      sitemapUrls.push(`${content.siteUrl}${lang.urlPrefix}${pageFile}`);
    }
  }

  const sitemap = `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${sitemapUrls
    .map((url) => `  <url><loc>${url}</loc></url>`)
    .join("\n")}\n</urlset>\n`;
  writeFileSync(join(DIST, "sitemap.xml"), sitemap);

  writeFileSync(
    join(DIST, "robots.txt"),
    `User-agent: *\nAllow: /\n\nSitemap: ${content.siteUrl}/sitemap.xml\n`,
  );

  const pageCount = locales.length * 2;
  console.log(`Built ${pageCount} pages for ${locales.length} locales into dist/`);
}

main();
