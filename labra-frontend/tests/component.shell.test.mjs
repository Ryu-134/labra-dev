import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';

const layoutSource = readFileSync('src/routes/+layout.svelte', 'utf8');
const headerSource = readFileSync('src/lib/components/header.svelte', 'utf8');

test('layout composes header and footer shell', () => {
  assert.equal(layoutSource.includes('<Header />'), true, 'layout should render Header');
  assert.equal(layoutSource.includes('<Footer />'), true, 'layout should render Footer');
});

test('header exposes sprint 1 nav and environment indicator', () => {
  assert.equal(headerSource.includes('/dashboard'), true, 'header should link to dashboard');
  assert.equal(headerSource.includes('/settings'), true, 'header should link to settings');
  assert.equal(headerSource.includes('Env:'), true, 'header should render environment indicator');
});
