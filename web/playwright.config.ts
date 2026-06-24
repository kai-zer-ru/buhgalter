import { defineConfig, devices } from '@playwright/test';

const port = process.env.BUHGALTER_E2E_PORT ?? '9876';
const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? `http://127.0.0.1:${port}`;

export default defineConfig({
	testDir: 'e2e',
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: process.env.CI ? 1 : 0,
	workers: 1,
	reporter: process.env.CI ? 'github' : 'list',
	timeout: 60_000,
	expect: { timeout: 20_000 },
	use: {
		baseURL,
		trace: 'on-first-retry',
		locale: 'ru-RU'
	},
	projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
	webServer: {
		command: `BUHGALTER_ADDR=:${port} bash ../scripts/e2e-server.sh`,
		url: `${baseURL}/api/v1/health`,
		reuseExistingServer: false,
		timeout: 120_000,
		cwd: import.meta.dirname
	}
});
