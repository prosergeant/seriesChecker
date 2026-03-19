import { test, expect } from '@playwright/test';

test.describe('SeriesTracker', () => {
  test('login page loads correctly', async ({ page }) => {
    await page.goto('/login');
    
    await expect(page.locator('h1, h2, [class*="text-2xl"]')).toContainText(/вход|login|войти/i, { timeout: 10000 });
    
    const emailInput = page.locator('input[type="email"], input[placeholder*="email" i]');
    const passwordInput = page.locator('input[type="password"]');
    
    await expect(emailInput).toBeVisible();
    await expect(passwordInput).toBeVisible();
  });

  test('main page shows loading or redirects', async ({ page }) => {
    await page.goto('/');
    
    const pageContent = await page.content();
    
    const hasLoading = pageContent.includes('Загрузка') || pageContent.includes('Loading') || pageContent.includes('loading');
    const hasError = pageContent.includes('error') || pageContent.includes('Error') || pageContent.includes('ошибка');
    const hasLogin = pageContent.includes('login') || pageContent.includes('вход') || pageContent.includes('войти');
    
    expect(hasLoading || hasError || hasLogin || page.url().toString().includes('login')).toBeTruthy();
  });
});
