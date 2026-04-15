import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';

const layoutSource = readFileSync('src/routes/+layout.svelte', 'utf8');
const headerSource = readFileSync('src/lib/components/header.svelte', 'utf8');
const loginSource = readFileSync('src/routes/login/+page.svelte', 'utf8');
const settingsSource = readFileSync('src/routes/settings/+page.svelte', 'utf8');
const appDetailsSource = readFileSync('src/routes/apps/[id]/+page.svelte', 'utf8');

test('layout composes header and footer shell', () => {
  assert.equal(layoutSource.includes('<Header />'), true, 'layout should render Header');
  assert.equal(layoutSource.includes('<Footer />'), true, 'layout should render Footer');
});

test('header exposes sprint 1 nav and environment indicator', () => {
  assert.equal(headerSource.includes('/dashboard'), true, 'header should link to dashboard');
  assert.equal(headerSource.includes('/settings'), true, 'header should link to settings');
  assert.equal(headerSource.includes('Env:'), true, 'header should render environment indicator');
});

test('sprint 2 auth and aws settings UI exists', () => {
  assert.equal(loginSource.includes('Create Session'), true, 'login page should create auth session');
  assert.equal(settingsSource.includes('Validate + Save'), true, 'settings page should save aws connection');
});

test('sprint 3 app details includes infra output and config history sections', () => {
  assert.equal(appDetailsSource.includes('Infra Outputs'), true, 'app details should show infra outputs');
  assert.equal(appDetailsSource.includes('Config History'), true, 'app details should show config history');
});
