import fs from 'node:fs';
import { test as setup } from '@playwright/test';
import { authFile, completeSetupIfNeeded, login } from './helpers/auth';

setup('authenticate', async ({ page }) => {
	await completeSetupIfNeeded(page);
	await login(page);
	fs.mkdirSync(authFile.replace(/[/\\][^/\\]+$/, ''), { recursive: true });
	await page.context().storageState({ path: authFile });
});
