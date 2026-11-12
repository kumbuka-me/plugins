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

  assert.match(first, /<svg\b/);
  assert.match(dark, /<svg\b/);
  assert.equal(first, again);
  assert.notEqual(first, dark);
});
