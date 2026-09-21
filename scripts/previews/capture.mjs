import { createServer } from "node:http";
import { mkdir, readdir, readFile, stat } from "node:fs/promises";
import {
  dirname,
  extname,
  isAbsolute,
  join,
  normalize,
  relative,
  resolve,
} from "node:path";
import { chromium } from "playwright";

const repository = process.env.PREVIEW_REPOSITORY;
const site = process.env.PREVIEW_SITE;
const browserChannel = (process.env.PREVIEW_BROWSER_CHANNEL || "").trim();

if (!repository || !site) {
  throw new Error("Missing preview capture environment.");
}

const contentTypes = new Map([
  [".css", "text/css; charset=utf-8"],
  [".html", "text/html; charset=utf-8"],
  [".ico", "image/x-icon"],
  [".js", "text/javascript; charset=utf-8"],
  [".json", "application/json; charset=utf-8"],
  [".png", "image/png"],
  [".svg", "image/svg+xml"],
  [".woff2", "font/woff2"],
]);

async function pluginDirectories() {
  const entries = await readdir(repository, { withFileTypes: true });
  const result = [];

  for (const entry of entries) {
    if (!entry.isDirectory()) continue;
    try {
      await stat(join(repository, entry.name, "plugin.yaml"));
      result.push(entry.name);
    } catch {
      // Ordinary repository directories are not plugins.
    }
  }

  return result.sort();
}

function safeSitePath(pathname) {
  const decoded = decodeURIComponent(pathname);
  const relativePath = decoded.replace(/^\/+/, "");
  const root = resolve(site);
  let filename = resolve(root, normalize(relativePath));
  const remainder = relative(root, filename);

  if (
    remainder === ".." ||
    remainder.startsWith("../") ||
    remainder.startsWith("..\\") ||
    isAbsolute(remainder)
  ) {
    return "";
  }
  if (decoded.endsWith("/")) {
    filename = join(filename, "index.html");
  }

  return filename;
}

async function responseFile(pathname) {
  let filename = safeSitePath(pathname);
  if (!filename) return null;

  try {
    const info = await stat(filename);
    if (info.isDirectory()) filename = join(filename, "index.html");
  } catch {
    if (!extname(filename)) filename = join(filename, "index.html");
  }

  try {
    return {
      filename,
      data: await readFile(filename),
    };
  } catch {
    return null;
  }
}

const server = createServer(async (request, response) => {
  const url = new URL(request.url || "/", "http://127.0.0.1");
  const file = await responseFile(url.pathname);

  if (!file) {
    response.writeHead(404, { "Content-Type": "text/plain; charset=utf-8" });
    response.end("Not found");
    return;
  }

  response.writeHead(200, {
    "Content-Type":
      contentTypes.get(extname(file.filename).toLowerCase()) ||
      "application/octet-stream",
    "Cache-Control": "no-store",
  });
  response.end(file.data);
});

await new Promise((resolveServer, rejectServer) => {
  server.once("error", rejectServer);
  server.listen(0, "127.0.0.1", resolveServer);
});

const address = server.address();
if (!address || typeof address === "string") {
  await new Promise((resolveServer) => server.close(resolveServer));
  throw new Error("Could not resolve preview server address.");
}
const baseURL = `http://127.0.0.1:${address.port}`;

const launchOptions = { headless: true };
if (browserChannel) launchOptions.channel = browserChannel;

let browser;

async function settle(page) {
  await page.evaluate(() => document.fonts.ready);
  await page.waitForTimeout(150);

  const frames = page.locator("iframe.kumbuka-plugin-frame");
  if ((await frames.count()) > 0) {
    await page.waitForFunction(
      () =>
        [...document.querySelectorAll("iframe.kumbuka-plugin-frame")].every(
          (frame) => frame.dataset.pluginReady === "true",
        ),
      undefined,
      { timeout: 15_000 },
    );
  }

  await page.waitForTimeout(100);
}

async function capturePreview(page, plugin) {
  const path = `/previews/${plugin}/`;
  const response = await page.goto(new URL(path, baseURL).toString(), {
    waitUntil: "networkidle",
  });
  if (!response || !response.ok()) {
    throw new Error(
      `Preview page ${path} returned HTTP ${response?.status() ?? "unknown"}.`,
    );
  }

  await settle(page);
  const output = join(repository, plugin, "assets", "preview.png");
  await mkdir(dirname(output), { recursive: true });
  await page.screenshot({
    path: output,
    animations: "disabled",
  });
}

try {
  browser = await chromium.launch(launchOptions);
  const plugins = await pluginDirectories();
  const context = await browser.newContext({
    viewport: { width: 1280, height: 720 },
    deviceScaleFactor: 1,
    colorScheme: "light",
    reducedMotion: "reduce",
    serviceWorkers: "block",
  });
  const page = await context.newPage();
  page.setDefaultTimeout(15_000);

  for (const plugin of plugins) {
    await capturePreview(page, plugin);
  }

  await context.close();
  console.log(`Generated ${plugins.length} plugin previews.`);
} finally {
  await browser?.close();
  await new Promise((resolveServer) => server.close(resolveServer));
}
