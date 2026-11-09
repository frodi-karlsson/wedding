import { test, expect } from '@playwright/test';

test('venue map renders with both pins and a Google Maps link', async ({ page }) => {
  await page.goto('/');
  await page.locator('#location').scrollIntoViewIfNeeded();

  // Leaflet initialised (assert structure, not external tiles)
  await expect(page.locator('.leaflet-container')).toBeVisible();
  await expect(page.locator('.map-pin')).toHaveCount(2);

  const cta = page.locator('.venue-map__cta');
  await expect(cta).toBeVisible();
  await expect(cta).toHaveAttribute('href', /google\.com\/maps\/dir/);
});

test('venue map labels each pin with its role', async ({ page }) => {
  await page.goto('/en');
  await page.locator('#location').scrollIntoViewIfNeeded();

  const labels = page.locator('.map-pin__label');

  await expect(labels).toHaveText(['Ceremony', 'Reception']);
  await expect(labels.first()).toBeVisible();
});
