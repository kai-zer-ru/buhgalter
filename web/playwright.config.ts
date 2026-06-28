import { defineConfig, devices } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const port = process.env.BUHGALTER_E2E_PORT ?? '9876';
const baseURL = process.env.PLAYWRIGHT_BASE_URL ?? `http://127.0.0.1:${port}`;
const __dirname = path.dirname(fileURLToPath(import.meta.url));
const authFile = path.join(__dirname, 'e2e/.auth/admin.json');

export default defineConfig({
	testDir: 'e2e',
	fullyParallel: false,
	forbidOnly: !!process.env.CI,
	retries: 1,
	workers: 1,
	reporter: process.env.CI ? 'github' : 'list',
	timeout: 60_000,
	expect: { timeout: 20_000 },
	use: {
		baseURL,
		trace: 'on-first-retry',
		locale: 'ru-RU'
	},
	projects: [
		{ name: 'setup', testMatch: /auth\.setup\.ts/ },
		{
			name: 'chromium',
			use: { ...devices['Desktop Chrome'], storageState: authFile },
			dependencies: ['setup'],
			testIgnore: /auth\.setup\.ts/
		}
	],
	webServer: {
		command: `BUHGALTER_ADDR=:${port} bash ../scripts/e2e-server.sh`,
		url: `${baseURL}/api/v1/health`,
		reuseExistingServer: false,
		timeout: 120_000,
		cwd: import.meta.dirname
	}
});
