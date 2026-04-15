import test from 'node:test';
import assert from 'node:assert/strict';
import { existsSync } from 'node:fs';

const requiredRoutes = [
  'src/routes/+page.svelte',
  'src/routes/login/+page.svelte',
  'src/routes/dashboard/+page.svelte',
  'src/routes/apps/+page.svelte',
  'src/routes/apps/[id]/+page.svelte',
  'src/routes/deploys/+page.svelte',
  'src/routes/deploys/[id]/+page.svelte',
  'src/routes/settings/+page.svelte'
];

test('route map includes Sprint 1 scaffold pages', () => {
  for (const routeFile of requiredRoutes) {
    assert.equal(existsSync(routeFile), true, `missing route file: ${routeFile}`);
  }
});
