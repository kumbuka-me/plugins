import test from 'node:test';
import assert from 'node:assert/strict';
import { readFile } from 'node:fs/promises';
import { chromium } from 'playwright';

// Real browser rendering ensures build-time artifacts use the bundled Mermaid engine.
test('Mermaid exports passive SVGs for both themes', async () => {
  const browser = await chromium.launch({ headless: true });
  try {
    const page = await browser.newPage();
    await page.setContent('<!doctype html><html><body></body></html>');
    await page.addScriptTag({ content: await readFile(new URL('../../mermaid/assets/mermaid.min.js', import.meta.url), 'utf8') });
    await page.addScriptTag({ content: await readFile(new URL('../../mermaid/assets/plugin.js', import.meta.url), 'utf8') });

    const variants = await page.evaluate(async () => {
      const root = document.createElement('div');
      const code = document.createElement('code');
      code.className = 'language-mermaid';
      code.textContent = 'flowchart LR\nA --> B';
      const pre = document.createElement('pre');
      pre.append(code);
      root.append(pre);
      document.body.append(root);
      const light = await globalThis.kumbukaPlugin.exportStatic(root, { theme: 'light' });
      root.replaceChildren(pre);
      const dark = await globalThis.kumbukaPlugin.exportStatic(root, { theme: 'dark' });
      root.remove();
      return { light, dark };
    });

    assert.match(variants.light.content, /<svg[\s>]/);
    assert.match(variants.dark.content, /<svg[\s>]/);
    assert.doesNotMatch(variants.light.content, /<foreignObject\b/i);
    assert.doesNotMatch(variants.dark.content, /<foreignObject\b/i);
    assert.notEqual(variants.light.content, variants.dark.content);
  } finally {
    await browser.close();
  }
});

test('Mermaid preserves interactive blocks for normal browser rendering', async () => {
  const browser = await chromium.launch({ headless: true });
  try {
    const page = await browser.newPage();
    await page.setContent('<!doctype html><html><body></body></html>');
    await page.addScriptTag({ content: await readFile(new URL('../../mermaid/assets/plugin.js', import.meta.url), 'utf8') });

    const result = await page.evaluate(async () => {
      const root = document.createElement('div');
      root.innerHTML = '<pre><code class="language-mermaid">%% kumbuka-interactive: true\nflowchart LR</code></pre>';
      return globalThis.kumbukaPlugin.exportStatic(root, { theme: 'light' });
    });

    assert.equal(result, null);
  } finally {
    await browser.close();
  }
});
