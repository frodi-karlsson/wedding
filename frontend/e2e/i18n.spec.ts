import { test, expect } from '@playwright/test';

test('Icelandic route renders translated copy', async ({ page }) => {
  await page.goto('/is/');
  // section heading translated (was an English placeholder before the i18n fix)
  await expect(page.locator('#location .section__heading')).toHaveText('Staðsetning');
});

test('language switch preserves query params and fragment', async ({ page }) => {
  await page.goto('/?id=abc123#location');

  // open the language dropdown and pick Icelandic
  await page.locator('.lang-dropdown__trigger').click();
  await page.locator('.lang-dropdown__item', { hasText: 'Íslenska' }).click();

  await page.waitForURL(/\/is(\/|\b)/);
  const url = new URL(page.url());
  expect(url.pathname).toMatch(/^\/is\/?$/);
  expect(url.searchParams.get('id')).toBe('abc123');
  expect(url.hash).toBe('#location');
});

test('root renders the default locale (Swedish)', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('html')).toHaveAttribute('lang', 'sv');
});

test('the /en route renders English', async ({ page }) => {
  await page.goto('/en');
  await expect(page.locator('html')).toHaveAttribute('lang', 'en');
});
