import { createHash } from "node:crypto";
import { cp, mkdir, readFile, readdir, writeFile } from "node:fs/promises";
import { homedir, platform } from "node:os";
import { dirname, isAbsolute, join } from "node:path";

const repository = process.env.PREVIEW_REPOSITORY;
const dist = process.env.PREVIEW_DIST;
const work = process.env.PREVIEW_WORK;
const selectedPlugins = new Set(
  (process.env.PREVIEW_PLUGINS || "")
    .split(/[\s,]+/)
    .map((value) => value.trim())
    .filter(Boolean),
);

if (!repository || !dist || !work) {
  throw new Error("Missing preview preparation environment.");
}

function topLevelValue(source, field) {
  const match = source.match(new RegExp(`^${field}:\\s*(.+?)\\s*$`, "m"));
  if (!match) throw new Error(`Missing ${field} in plugin manifest.`);
  return match[1].trim().replace(/^["']|["']$/g, "");
}

function goUserCacheDir() {
  const home = process.env.HOME || homedir();

  if (platform() === "darwin") {
    return join(home, "Library", "Caches");
  }
  if (platform() === "win32") {
    const local = process.env.LocalAppData;
    if (!local) throw new Error("LocalAppData is not set.");
    return local;
  }

  const xdg = process.env.XDG_CACHE_HOME;
  if (xdg && isAbsolute(xdg)) return xdg;
  return join(home, ".cache");
}

async function pluginDirectories() {
  const entries = await readdir(repository, { withFileTypes: true });
  const result = [];

  for (const entry of entries) {
    if (!entry.isDirectory()) continue;

    try {
      await readFile(join(repository, entry.name, "plugin.yaml"), "utf8");
      if (selectedPlugins.size === 0 || selectedPlugins.has(entry.name)) {
        result.push(entry.name);
      }
    } catch {
      // Ordinary repository directories are not plugins.
    }
  }

  return result.sort();
}

async function previewSource(plugin, pluginDir) {
  const previewPath = join(pluginDir, "preview.md");
  const preview = await readFile(previewPath, "utf8");
  if (preview.trim() === "") {
    throw new Error(`${plugin}/preview.md must not be empty.`);
  }

  const staticPath = join(pluginDir, "preview.static.md");
  try {
    const staticPreview = await readFile(staticPath, "utf8");
    if (staticPreview.trim() === "") {
      throw new Error(`${plugin}/preview.static.md must not be empty.`);
    }
    return staticPreview;
  } catch (error) {
    if (error?.code !== "ENOENT") throw error;
  }

  return preview;
}

async function writeSupportPages(plugin, sourceDir) {
  if (plugin === "subpages") {
    await writeFile(
      join(sourceDir, "getting-started.md"),
      `# Getting started

Prepare the service and confirm access before making changes.
`,
    );
    await writeFile(
      join(sourceDir, "operations.md"),
      `# Operations

Day-two procedures for running the service safely.
`,
    );
    await writeFile(
      join(sourceDir, "troubleshooting.md"),
      `# Troubleshooting

Common symptoms, checks, and recovery steps.
`,
    );
  }
}

const plugins = await pluginDirectories();
if (plugins.length === 0) throw new Error("No plugin manifests found.");

const cacheRoot = join(goUserCacheDir(), "kumbuka", "plugins");
const configsDir = join(work, "configs");
const dependenciesDir = join(work, "plugins");
const sourcesDir = join(work, "sources");
const sitesDir = join(work, "sites");

await mkdir(configsDir, { recursive: true });
await mkdir(dependenciesDir, { recursive: true });
await mkdir(sourcesDir, { recursive: true });
await mkdir(sitesDir, { recursive: true });

for (const plugin of plugins) {
  const pluginDir = join(repository, plugin);
  const manifest = await readFile(join(pluginDir, "plugin.yaml"), "utf8");
  const preview = await previewSource(plugin, pluginDir);

  const id = topLevelValue(manifest, "id");
  const version = topLevelValue(manifest, "version");
  const name = topLevelValue(manifest, "name");
  const sourceDir = join(sourcesDir, plugin);
  const siteDir = join(sitesDir, plugin);

  await mkdir(sourceDir, { recursive: true });
  await writeFile(join(sourceDir, "index.md"), preview);
  await writeSupportPages(plugin, sourceDir);

  const tagPrefix = `${plugin}/v`;
  const asset = plugin;
  const dependencies = `format = 1

[[plugin]]
id = ${JSON.stringify(id)}
repository = "kumbuka-me/plugins"
tag_prefix = ${JSON.stringify(tagPrefix)}
asset = ${JSON.stringify(asset)}
version = ${JSON.stringify(version)}
`;
  await writeFile(join(dependenciesDir, `${plugin}.toml`), dependencies);

  const cacheKey = createHash("sha256")
    .update(`kumbuka-me/plugins\n${tagPrefix}\n${asset}`)
    .digest("hex");
  const cacheFile = join(
    cacheRoot,
    cacheKey,
    id,
    version,
    "plugin.kumbukaplugin",
  );
  await mkdir(dirname(cacheFile), { recursive: true });

  const packageFile = join(dist, `${plugin}-${version}.kumbukaplugin`);
  await cp(packageFile, cacheFile);

  const config = `site_name = ${JSON.stringify(name)}
site_url = ${JSON.stringify(`http://127.0.0.1/${plugin}/`)}
source_dir = ${JSON.stringify(sourceDir)}
output_dir = ${JSON.stringify(siteDir)}
theme = "Light"
language = "en"
navigation_style = "sidebar"
navigation_density = "comfortable"
expand_content_when_hidden = true
sidebar_width = 280
robots = "none"
`;
  await writeFile(join(configsDir, `${plugin}.toml`), config);
}

await writeFile(join(work, "plugins.txt"), `${plugins.join("\n")}\n`);
console.log(`Prepared ${plugins.length} plugin preview pages.`);
