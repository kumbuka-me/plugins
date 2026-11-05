import test from 'node:test';
import assert from 'node:assert/strict';
import {readFile} from 'node:fs/promises';
import {chromium} from 'playwright';

async function editor(name, source, check) {
  const browser = await chromium.launch({channel:process.env.BROWSER_CHANNEL || undefined,headless:true});
  try {
    const page = await browser.newPage();
    const script = await readFile(new URL(`../../${name}/assets/plugin.js`,import.meta.url),'utf8');
    await page.route('http://plugin.test/**', route => route.fulfill({contentType:'text/html',body:'<main></main>'}));
    await page.goto('http://plugin.test/');
    await page.addScriptTag({content:script});
    await page.evaluate(source => { globalThis.kumbukaPlugin.edit(document.querySelector('main'), {source,theme:'light',onChange(value){globalThis.draft=value},onValidity(state){globalThis.validity=state}}); },source);
    await check(page);
  } finally {await browser.close();}
}

test('Runbook retains structured controls through invalid titles and choices', async()=>{
  await editor('runbook','start: Choose\n  Yes -> end\n  No -> end\nend: Finish',async page=>{
    assert.equal(await page.evaluate(()=>globalThis.validity.valid),true);
    await page.getByLabel('Title for start').fill('');
    await page.getByLabel('Title for start').press('Tab');
    assert.equal(await page.locator('fieldset').count(),2);
    assert.equal(await page.evaluate(()=>globalThis.validity.valid),false);
    await page.getByLabel('Title for start').fill('Repaired');
    await page.getByLabel('Title for start').press('Tab');
    assert.equal(await page.evaluate(()=>globalThis.validity.valid),true);
    await page.getByLabel('Choice label from start').first().fill('');
    await page.getByLabel('Choice label from start').first().press('Tab');
    assert.equal(await page.locator('fieldset').count(),2);
    assert.equal(await page.evaluate(()=>globalThis.validity.valid),false);
    await page.getByLabel('Choice label from start').first().fill('Yes');
    await page.getByLabel('Choice label from start').first().press('Tab');
    assert.equal(await page.evaluate(()=>globalThis.validity.valid),true);
    assert.equal(await page.locator('textarea').count(),0);
    await page.getByRole('button',{name:'Add answer'}).first().click();
    assert.equal(await page.getByLabel('Choice label from start').count(),3);
    assert.equal(await page.evaluate(()=>globalThis.validity.valid),true);
    assert.equal(await page.locator('.answer-link').count(),3);
  });
});

test('Mermaid owns alignment editing and rendering of saved directives', async()=>{
  await editor('mermaid','%% kumbuka-align: LEFT\r\nflowchart LR\nA --> B',async page=>{
    const alignment=page.getByLabel('Diagram alignment');
    assert.equal(await alignment.inputValue(),'left');
    await alignment.selectOption('right');
    assert.equal(await page.evaluate(()=>globalThis.draft),'%% kumbuka-align: right\nflowchart LR\nA --> B');
    await alignment.selectOption('center');
    assert.equal(await page.evaluate(()=>globalThis.draft),'flowchart LR\nA --> B');
    await page.evaluate(async()=>{
      globalThis.mermaid={initialize(){},async run(){}};
      await globalThis.kumbukaPlugin.render(document.querySelector('main'),{source:'%% kumbuka-align: right\nflowchart LR',theme:'light'});
    });
    assert.equal(await page.locator('.mermaid').evaluate(el=>el.style.textAlign),'right');
  });
});
