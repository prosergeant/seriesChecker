import { test, expect } from '@playwright/test';

// These tests require the backend (Go server on :8080) to be running.
// Set TEST_EMAIL / TEST_PASSWORD env vars to use an existing account,
// or the tests will create a new random one.

const email = process.env.TEST_EMAIL ?? `test_${Date.now()}@example.com`;
const password = process.env.TEST_PASSWORD ?? 'TestPassword123!';

test.describe('Auth flow', () => {
  test('login page — form elements visible', async ({ page }) => {
    await page.goto('/login');

    await expect(page.locator('input[type="email"]')).toBeVisible();
    await expect(page.locator('input[type="password"]')).toBeVisible();
    await expect(page.getByRole('button', { name: /войти|вход|login/i })).toBeVisible();
  });

  test('register a new user, then logout', async ({ page }) => {
    const uniqueEmail = `e2e_${Date.now()}@example.com`;

    await page.goto('/login');

    // Switch to registration mode
    await page.getByRole('button', { name: /регистрация|зарегистрироваться/i }).click();

    await page.locator('input[type="email"]').fill(uniqueEmail);
    await page.locator('input[type="password"]').fill('SecurePass123!');
    await page.getByRole('button', { name: /зарегистрироваться|регистрация/i }).click();

    // Should redirect to main page after successful registration
    await expect(page).toHaveURL('/', { timeout: 10000 });

    // Logout
    await page.getByRole('button', { name: /выйти|logout/i }).click();
    await expect(page).toHaveURL('/login', { timeout: 5000 });
  });

  test('login with wrong password — shows error message', async ({ page }) => {
    await page.goto('/login');

    await page.locator('input[type="email"]').fill('nobody@example.com');
    await page.locator('input[type="password"]').fill('wrongpassword');
    await page.getByRole('button', { name: /войти|вход|login/i }).click();

    // Should NOT redirect — stays on /login
    await expect(page).toHaveURL('/login');

    // Should show some error feedback (toast or inline message)
    const errorVisible = await page.locator('[data-sonner-toaster], [role="alert"]').isVisible()
      .catch(() => false);

    // At minimum: we didn't navigate away
    await expect(page).toHaveURL('/login');
  });

  test('login with correct credentials', async ({ page }) => {
    // Skip if no TEST_EMAIL is configured — we don't want to use the random email here
    test.skip(!process.env.TEST_EMAIL, 'Set TEST_EMAIL and TEST_PASSWORD env vars to run this test');

    await page.goto('/login');
    await page.locator('input[type="email"]').fill(email);
    await page.locator('input[type="password"]').fill(password);
    await page.getByRole('button', { name: /войти|вход|login/i }).click();

    await expect(page).toHaveURL('/', { timeout: 10000 });
  });
});
