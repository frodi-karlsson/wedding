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
