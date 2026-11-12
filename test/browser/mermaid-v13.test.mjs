import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {chromium} from 'playwright';

// servePlugin supplies only the two JavaScript assets included in the released plugin.
async function servePlugin(page) {
  await page.route('http://mermaid-plugin.test/**', async route => {
    const pathname = new URL(route.request().url()).pathname;
    if (pathname === '/') {
      await route.fulfill({contentType: 'text/html', body: '<main></main>'});
      return;
    }

    const filename = pathname === '/assets/plugin.js'
      ? '../../mermaid/assets/plugin.js'
      : pathname === '/assets/mermaid.min.js'
        ? '../../mermaid/assets/mermaid.min.js'
        : null;
    if (!filename) {
      await route.fulfill({status: 404});
      return;
    }
    await route.fulfill({
      contentType: 'text/javascript',
      body: await readFile(new URL(filename, import.meta.url)),
    });
  });

  await page.goto('http://mermaid-plugin.test/');
  await page.addScriptTag({url: 'http://mermaid-plugin.test/assets/plugin.js'});
}

test('Mermaid v13 alpha renders flowchart SVG using packaged JavaScript', async () => {
  const bundle = await readFile(new URL('../../mermaid/assets/mermaid.min.js', import.meta.url), 'utf8');
  assert.ok(bundle.startsWith('// Kumbuka Mermaid bundle: 13.0.0-alpha.3\n'));

  const browser = await chromium.launch({
    channel: process.env.BROWSER_CHANNEL || undefined,
    headless: true,
  });
  try {
    const page = await browser.newPage();
    await servePlugin(page);

    await page.evaluate(async () => {
      await globalThis.kumbukaPlugin.render(document.querySelector('main'), {
        source: 'flowchart LR\n  A[Start] --> B[Finish]',
        theme: 'light',
      });
    });

    assert.equal(await page.locator('.mermaid svg').count(), 1);
    assert.equal(await page.locator('.mermaid').evaluate(el => el.classList.contains('mermaid')), true);
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
    await servePlugin(page);

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
