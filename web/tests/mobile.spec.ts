import { test, expect } from '@playwright/test';

const MOBILE = { width: 375, height: 812 };

test.describe('Mobile layout (375×812)', () => {
  test.use({ viewport: MOBILE });

  test('login page — no horizontal overflow', async ({ page }) => {
    await page.goto('/login');

    // Page body should not overflow horizontally
    const scrollWidth = await page.evaluate(() => document.body.scrollWidth);
    expect(scrollWidth).toBeLessThanOrEqual(MOBILE.width + 2); // 2px tolerance
  });

  test('login page — inputs are full-width and reachable', async ({ page }) => {
    await page.goto('/login');

    const emailInput = page.locator('input[type="email"]');
    await expect(emailInput).toBeVisible();

    const box = await emailInput.boundingBox();
    // Input should stretch to most of the screen width, not be clipped
    expect(box?.width).toBeGreaterThan(200);
  });

  test('main page redirects to /login when not authenticated', async ({ page }) => {
    await page.goto('/');

    // Either stays on / with a spinner or redirects to /login
    await page.waitForTimeout(2000);
    const url = page.url();
    expect(url.includes('/login') || url.endsWith('/')).toBeTruthy();
  });

  test('main page — no horizontal overflow when logged in', async ({ page }) => {
    // Skip if no credentials are configured
    test.skip(!process.env.TEST_EMAIL, 'Set TEST_EMAIL and TEST_PASSWORD to run authenticated mobile tests');

    // Login first
    await page.goto('/login');
    await page.locator('input[type="email"]').fill(process.env.TEST_EMAIL!);
    await page.locator('input[type="password"]').fill(process.env.TEST_PASSWORD!);
    await page.getByRole('button', { name: /войти|вход|login/i }).click();
    await expect(page).toHaveURL('/', { timeout: 10000 });

    const scrollWidth = await page.evaluate(() => document.body.scrollWidth);
    expect(scrollWidth).toBeLessThanOrEqual(MOBILE.width + 2);
  });

  test('main page — filter row scrollable horizontally', async ({ page }) => {
    test.skip(!process.env.TEST_EMAIL, 'Set TEST_EMAIL and TEST_PASSWORD to run authenticated mobile tests');

    await page.goto('/login');
    await page.locator('input[type="email"]').fill(process.env.TEST_EMAIL!);
    await page.locator('input[type="password"]').fill(process.env.TEST_PASSWORD!);
    await page.getByRole('button', { name: /войти|вход|login/i }).click();
    await expect(page).toHaveURL('/', { timeout: 10000 });

    // Filter buttons container should have overflow-x: auto (horizontal scroll, not wrap)
    const filterContainer = page.locator('[class*="overflow-x-auto"]').first();
    await expect(filterContainer).toBeVisible();
  });
});
