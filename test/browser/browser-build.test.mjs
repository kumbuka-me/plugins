import test from 'node:test';
import assert from 'node:assert/strict';
import {execFileSync} from 'node:child_process';
import {copyFile, chmod, mkdtemp, mkdir, readFile, rm, writeFile} from 'node:fs/promises';
import {tmpdir} from 'node:os';
import {join} from 'node:path';

// The build must discover plugin-owned bundlers even when an archive drops their executable bits.
test('browser build invokes a non-executable plugin bundler and its updater', async () => {
  const directory = await mkdtemp(join(tmpdir(), 'kumbuka-browser-build-'));
  const plugin = join(directory, 'mermaid');

  try {
    await mkdir(join(directory, 'scripts', 'build'), {recursive: true});
    await mkdir(join(plugin, 'assets'), {recursive: true});

    await copyFile(
      new URL('../../scripts/build/browser.sh', import.meta.url),
      join(directory, 'scripts', 'build', 'browser.sh'),
    );
    await copyFile(
      new URL('../../mermaid/browser-build.sh', import.meta.url),
      join(plugin, 'browser-build.sh'),
    );
    await writeFile(join(plugin, 'browser.ts'), '// Test browser module\n');
    await writeFile(join(plugin, 'headless.ts'), '// Test headless module\n');
    await writeFile(join(plugin, 'update-assets.sh'), `#!/bin/sh
set -eu
root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
printf '// Kumbuka Mermaid browser bundle: 13.0.0-alpha.3\n' > "$root/assets/plugin.js"
printf '// Kumbuka Mermaid headless bundle: 13.0.0-alpha.3\n' > "$root/assets/headless.js"
`);
    await chmod(join(plugin, 'browser-build.sh'), 0o644);
    await chmod(join(plugin, 'update-assets.sh'), 0o644);

    execFileSync('sh', ['scripts/build/browser.sh', 'mermaid'], {cwd: directory});

    assert.match(await readFile(join(plugin, 'assets', 'plugin.js'), 'utf8'), /Kumbuka Mermaid browser bundle/);
    assert.match(await readFile(join(plugin, 'assets', 'headless.js'), 'utf8'), /Kumbuka Mermaid headless bundle/);
  } finally {
    await rm(directory, {recursive: true, force: true});
  }
});
