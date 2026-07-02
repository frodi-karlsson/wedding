import { test, expect } from '@playwright/test';

test('nav link scrolls to its section', async ({ page }) => {
  await page.goto('/');
  await page.locator('.nav__link[href="#location"]').click();
  await expect(page).toHaveURL(/#location$/);
  await expect(page.locator('#location')).toBeInViewport();
});

test('RSVP nav link is gated on an invite id', async ({ page }) => {
  await page.goto('/');
  await expect(page.locator('.nav__link--rsvp')).toHaveCount(0);

  await page.goto('/?id=abc123');
  await expect(page.locator('.nav__link--rsvp')).toBeVisible();
});

test('mobile menu opens and closes', async ({ page }) => {
  await page.setViewportSize({ width: 375, height: 780 });
  await page.goto('/');

  const toggle = page.locator('.nav__toggle');
  const panel = page.locator('.nav__panel');

  await expect(toggle).toBeVisible();
  await toggle.click();
  await expect(panel).toHaveAttribute('data-open', '');

  // clicking a link closes the overlay
  await page.locator('.nav__link[href="#gifts"]').click();
  await expect(panel).not.toHaveAttribute('data-open', '');
});
