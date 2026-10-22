import fs from 'node:fs';
import { expect, test as setup } from '@playwright/test';
import { authFile, completeSetupIfNeeded, loginViaAPI } from './helpers/auth';

setup('authenticate', async ({ page }) => {
	await completeSetupIfNeeded(page);
	await loginViaAPI(page);
	const secretRes = await page.request.put('/api/v1/admin/settings/notification-secret', {
		data: { notification_secret_key: '12345678901234567890123456789012' }
	});
	expect(secretRes.ok(), `notification secret setup failed: ${secretRes.status()}`).toBeTruthy();
	fs.mkdirSync(authFile.replace(/[/\\][^/\\]+$/, ''), { recursive: true });
	await page.context().storageState({ path: authFile });
});
