import { createHash } from "node:crypto";
import {
  cp,
  mkdir,
  readFile,
  readdir,
  writeFile,
} from "node:fs/promises";
import { homedir, platform } from "node:os";
import { dirname, isAbsolute, join } from "node:path";

const repository = process.env.PREVIEW_REPOSITORY;
const dist = process.env.PREVIEW_DIST;
const work = process.env.PREVIEW_WORK;

if (!repository || !dist || !work) {
  throw new Error("Missing preview preparation environment.");
}

const liveExamplePlugins = new Set([
  "autolinks",
  "callouts",
  "definition-lists",
  "details",
  "footnotes",
  "mermaid",
  "strikethrough",
  "syntax-highlighting",
  "tables",
  "tabs",
  "task-lists",
  "typographer",
]);

function topLevelValue(source, field) {
  const match = source.match(new RegExp(`^${field}:\\s*(.+?)\\s*$`, "m"));
  if (!match) throw new Error(`Missing ${field} in plugin manifest.`);
  return match[1].trim().replace(/^["']|["']$/g, "");
}

function firstMarkdownFence(source) {
  const lines = source.split(/\r?\n/);

  for (let index = 0; index < lines.length; index += 1) {
    const opening = lines[index].match(/^(`{3,}|~{3,})\s*(markdown|md)\s*$/i);
    if (!opening) continue;

    const marker = opening[1][0];
    const width = opening[1].length;
    const content = [];

    for (let cursor = index + 1; cursor < lines.length; cursor += 1) {
      const closing = lines[cursor].match(/^(`+|~+)\s*$/);
      if (
        closing &&
        closing[1][0] === marker &&
        closing[1].length >= width
      ) {
        return `${content.join("\n").trim()}\n`;
      }
      content.push(lines[cursor]);
    }
  }

  return "";
}

function wikiTargets(source) {
  const targets = new Set();

  for (const match of source.matchAll(/\[\[([^\]\n]+)\]\]/g)) {
    let target = match[1].split("|", 1)[0].trim();
    target = target.split("#", 1)[0].trim();
    if (target) targets.add(target);
  }

  return targets;
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
      result.push(entry.name);
    } catch {
      // Ordinary repository directories are not plugins.
    }
  }

  return result.sort();
}

const plugins = await pluginDirectories();
if (plugins.length === 0) throw new Error("No plugin manifests found.");

const sourceDir = join(work, "source");
const siteDir = join(work, "site");
await mkdir(sourceDir, { recursive: true });

const index = [
  "# Kumbuka Plugins",
  "",
  "Rendered from the plugin READMEs in the current checkout.",
  "",
];

const dependencies = ["format = 1", ""];
const cacheRoot = join(goUserCacheDir(), "kumbuka", "plugins");
const requiredWikiTargets = new Map();

for (const plugin of plugins) {
  const pluginDir = join(repository, plugin);
  const manifest = await readFile(join(pluginDir, "plugin.yaml"), "utf8");
  const readme = await readFile(join(pluginDir, "README.md"), "utf8");
  for (const target of wikiTargets(readme)) {
    if (!requiredWikiTargets.has(target)) requiredWikiTargets.set(target, plugin);
  }
  const id = topLevelValue(manifest, "id");
  const version = topLevelValue(manifest, "version");
  const name = topLevelValue(manifest, "name");
  const description = topLevelValue(manifest, "description");

  const documentationDir = join(sourceDir, plugin);
  await mkdir(documentationDir, { recursive: true });
  await writeFile(join(documentationDir, "index.md"), readme);

  const previewDir = join(sourceDir, "previews", plugin);
  await mkdir(previewDir, { recursive: true });

  let example = "";
  if (liveExamplePlugins.has(plugin)) {
    example = firstMarkdownFence(readme);
  }

  const previewSource = example
    ? `# ${name}\n\n${description}\n\n## Example\n\n${example}`
    : readme;
  await writeFile(join(previewDir, "index.md"), previewSource);

  index.push(`- [${name}](${plugin}/)`);

  const tagPrefix = `${plugin}/v`;
  const asset = plugin;
  dependencies.push(
    "[[plugin]]",
    `id = ${JSON.stringify(id)}`,
    'repository = "kumbuka-me/plugins"',
    `tag_prefix = ${JSON.stringify(tagPrefix)}`,
    `asset = ${JSON.stringify(asset)}`,
    `version = ${JSON.stringify(version)}`,
    "",
  );

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
}

for (const [target, plugin] of [...requiredWikiTargets].sort(([left], [right]) =>
  left.localeCompare(right),
)) {
  const digest = createHash("sha256").update(target).digest("hex").slice(0, 12);
  await writeFile(
    join(sourceDir, plugin, `preview-target-${digest}.md`),
    `# ${target}\n\nGenerated only to resolve wiki-link examples while rendering plugin previews.\n`,
  );
}

await writeFile(join(sourceDir, "index.md"), `${index.join("\n")}\n`);
await writeFile(join(work, ".kumbukaplugins"), `${dependencies.join("\n")}\n`);

const config = `site_name = "Kumbuka Plugins"
site_url = "http://127.0.0.1/"
source_dir = ${JSON.stringify(sourceDir)}
output_dir = ${JSON.stringify(siteDir)}
theme = "Light"
language = "en"
navigation_style = "sidebar"
navigation_density = "comfortable"
expand_content_when_hidden = true
sidebar_width = 280
robots = "none"

[[external_links]]
label = "GitHub"
url = "https://github.com/kumbuka-me/plugins"
icon = "github-simple"
description = "Plugins"
`;

await writeFile(join(work, "kumbuka-site.toml"), config);
console.log(`Prepared ${plugins.length} plugin documentation pages.`);
