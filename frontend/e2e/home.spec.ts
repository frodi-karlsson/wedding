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
  for (const id of ['#welcome', '#location', '#schedule', '#dress', '#speeches', '#toastmasters', '#language', '#gifts', '#contact']) {
    await expect(page.locator(id)).toBeAttached();
  }
});

test('toastmasters section names them and links their address', async ({ page }) => {
  await page.goto('/en');

  const section = page.locator('#toastmasters');

  await expect(section).toContainText('Siri Andersson');
  await expect(section).toContainText('Tobias Gummeson');
  await expect(section.locator('a[href="mailto:toast@carlaochfrodi.wedding"]')).toHaveText('toast@carlaochfrodi.wedding');
  await expect(page.locator('#contact a[href="mailto:toast@carlaochfrodi.wedding"]')).toHaveCount(1);
});

test('speeches section asks for texts by the deadline', async ({ page }) => {
  await page.goto('/en');

  await expect(page.locator('#speeches')).toContainText('by 1 March 2027');
});

test('schedule lists the day in order', async ({ page }) => {
  await page.goto('/en');

  const times = await page.locator('#schedule .schedule__time').allTextContents();

  expect(times).toEqual(['approx. 14:45', '15:00', '16:00', '17:00–20:00', 'from 20:00']);
});
