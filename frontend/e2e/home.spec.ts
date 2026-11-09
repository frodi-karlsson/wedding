import { test, expect } from '@playwright/test';

test('landing page renders the hero and coming-soon copy', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('.hero__names')).toHaveText('Carla & Fróði');
  await expect(page.locator('.hero__eyebrow')).toBeVisible();
  await expect(page.locator('.hero__date')).toContainText('2027');
});

test('site is marked noindex', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('meta[name="robots"]')).toHaveAttribute(
    'content',
    /noindex/,
  );
});

test('all content sections are present', async ({ page }) => {
  await page.goto('/');
  for (const id of ['#welcome', '#location', '#dress', '#speeches', '#language', '#gifts', '#contact']) {
    await expect(page.locator(id)).toBeAttached();
  }
});

test('speeches section links the address to send speeches to', async ({ page }) => {
  await page.goto('/en');

  const link = page.locator('#speeches a[href="mailto:toast@carlaochfrodi.wedding"]');

  await expect(link).toHaveText('toast@carlaochfrodi.wedding');
  await expect(page.locator('#speeches')).toContainText('by 1 March 2027');
});
