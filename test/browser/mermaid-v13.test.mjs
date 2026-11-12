import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {chromium} from 'playwright';

// loadPlugin injects the packaged classic script as a blob URL, without
// relying on network routing or a separately hosted Mermaid runtime.
async function loadPlugin(page) {
  const script = await readFile(new URL('../../mermaid/assets/plugin.js', import.meta.url), 'utf8');
  await page.setContent('<main></main>');

  await page.evaluate(async source => {
    const url = URL.createObjectURL(new Blob([source], {type: 'text/javascript'}));
    try {
      await new Promise((resolve, reject) => {
        const element = document.createElement('script');
        element.src = url;
        element.onload = resolve;
        element.onerror = () => reject(new Error('packaged Mermaid script failed to load'));
        document.head.append(element);
      });
    } finally {
      URL.revokeObjectURL(url);
    }
  }, script);

  assert.equal(await page.evaluate(() => typeof globalThis.kumbukaPlugin?.render), 'function',
    'packaged Mermaid script did not register its browser renderer');
}

test('Mermaid v13 alpha renders flowchart SVG using packaged JavaScript', async () => {
  const bundle = await readFile(new URL('../../mermaid/assets/plugin.js', import.meta.url), 'utf8');
  assert.ok(bundle.startsWith('// Kumbuka Mermaid browser bundle: 13.0.0-alpha.3\n'));

  const browser = await chromium.launch({
    channel: process.env.BROWSER_CHANNEL || undefined,
    headless: true,
  });
  try {
    const page = await browser.newPage();
    const pageErrors = [];
    const unexpectedRequests = [];
    page.on('pageerror', error => pageErrors.push(error.message));
    page.on('request', request => {
      if (request.url().endsWith('/mermaid.min.js')) unexpectedRequests.push(request.url());
    });
    await loadPlugin(page);

    await page.evaluate(async () => {
      await globalThis.kumbukaPlugin.render(document.querySelector('main'), {
        source: 'flowchart LR\n  A[Start] --> B[Finish]',
        theme: 'light',
      });
    });

    assert.deepEqual(pageErrors, []);
    assert.deepEqual(unexpectedRequests, []);
    assert.equal(await page.locator('div.mermaid svg').count(), 1);
    assert.equal(await page.locator('div.mermaid').evaluate(el => el.style.textAlign), 'center');
  } finally {
    await browser.close();
  }
});

test('Mermaid visual editor keeps alignment and removes static-only controls', async () => {
  const browser = await chromium.launch({
    channel: process.env.BROWSER_CHANNEL || undefined,
    headless: true,
  });
  try {
    const page = await browser.newPage();
    await loadPlugin(page);

    await page.evaluate(() => {
      globalThis.kumbukaPlugin.edit(document.querySelector('main'), {
        source: '%% kumbuka-align: left\nflowchart LR\nA --> B',
        onChange(value) {globalThis.editedSource = value;},
      });
    });

    const alignment = page.getByLabel('Diagram alignment');
    assert.equal(await alignment.inputValue(), 'left');
    await alignment.selectOption('right');
    assert.equal(await page.evaluate(() => globalThis.editedSource),
      '%% kumbuka-align: right\nflowchart LR\nA --> B');
    assert.equal(await page.getByLabel('Interactive diagram').count(), 0);
  } finally {
    await browser.close();
  }
});
