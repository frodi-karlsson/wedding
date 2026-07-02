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
  for (const id of ['#welcome', '#location', '#dress', '#speeches', '#gifts', '#contact']) {
    await expect(page.locator(id)).toBeAttached();
  }
});
