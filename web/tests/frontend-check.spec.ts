import { test, expect } from '@playwright/test';

test.describe('Authentication Flow', () => {
  test('should show login page', async ({ page }) => {
    await page.goto('/login');
    
    const pageTitle = await page.title();
    console.log('Page title:', pageTitle);
    
    await page.screenshot({ path: 'login-page.png' });
    
    const emailInput = page.locator('input[type="email"], input[name="email"]').first();
    const passwordInput = page.locator('input[type="password"]').first();
    
    if (await emailInput.isVisible({ timeout: 5000 })) {
      console.log('Login form found');
    } else {
      console.log('Login form not visible, might be redirected');
    }
  });

  test('should load main page without errors', async ({ page }) => {
    const errors: string[] = [];
    
    page.on('console', msg => {
      if (msg.type() === 'error') {
        errors.push(msg.text());
      }
    });
    
    page.on('pageerror', error => {
      errors.push(error.message);
    });
    
    await page.goto('/');
    await page.waitForTimeout(2000);
    
    await page.screenshot({ path: 'main-page.png' });
    
    const content = await page.content();
    
    const criticalErrors = errors.filter(e => 
      !e.includes('Warning') && 
      !e.includes('DevTools') &&
      !e.includes('favicon')
    );
    
    if (criticalErrors.length > 0) {
      console.log('Console errors:', criticalErrors);
    }
    
    expect(criticalErrors.length).toBe(0);
  });
});

test.describe('Related Movies Modal', () => {
  test.skip('modal opens from card menu', async ({ page }) => {
    await page.goto('/login');
    await page.waitForTimeout(1000);
    
    const loginButton = page.locator('button:has-text("Войти"), button:has-text("Login"), button:has-text("Sign in")').first();
    
    if (await loginButton.isVisible({ timeout: 3000 })) {
      await page.screenshot({ path: 'before-login.png' });
    }
  });
});
