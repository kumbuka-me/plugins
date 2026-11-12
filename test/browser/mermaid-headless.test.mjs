import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import path from 'node:path';
import test from 'node:test';

const source = path.resolve('mermaid/assets/headless.js');

test('Mermaid v13 renders passive SVG with no DOM and deterministic IDs', async () => {
  const bundle = await fs.readFile(source, 'utf8');
  assert.match(bundle, /Kumbuka Mermaid headless bundle: 13\.0\.0-alpha\.3/);
  assert.equal(typeof globalThis.document, 'undefined');

  // The bundle owns the same API exposed to the Go embedded JS runtime.
  (0, eval)(bundle);
  assert.equal(typeof globalThis.kumbukaRenderArtifact, 'function');
  const diagram = 'flowchart LR\n A[Start] --> B[End]';
  const first = await globalThis.kumbukaRenderArtifact(diagram, 'default');
  const again = await globalThis.kumbukaRenderArtifact(diagram, 'default');
  const dark = await globalThis.kumbukaRenderArtifact(diagram, 'dark');

  for (const svg of [first, dark]) {
    const root = svg.match(/^<svg[^>]+>/)[0];
    const naturalWidth = root.match(/viewBox="0 0 ([\d.]+) [\d.]+"/)[1];
    assert.equal(root.match(/ width="([^"]+)"/)[1], naturalWidth);
  }
  assert.match(dark, /<svg\b/);
  assert.equal(first, again);
  assert.notEqual(first, dark);
});
