import type { Page } from '@playwright/test';
import { apiJSON } from './auth';

/** 54-FZ QR string. POST /receipts/parse ignores the body and returns the fixture receipt. */
const E2E_FISCAL_QR = 't=20260115T1030&s=150.00&fn=7288940700123456&i=12345&fp=9876543210&n=1';

/**
 * Photo upload decodes the QR in the page (`BarcodeDetector`) and POSTs `{ qr }` to
 * `/api/v1/receipts/parse`. Call before navigation so the stub is installed.
 */
export async function mockReceiptParseRoute(page: Page, scannedReceipt: Record<string, unknown>) {
	await page.addInitScript((qr: string) => {
		class BarcodeDetector {
			detect(): Promise<{ rawValue: string }[]> {
				return Promise.resolve([{ rawValue: qr }]);
			}
		}
		(globalThis as unknown as { BarcodeDetector: typeof BarcodeDetector }).BarcodeDetector =
			BarcodeDetector;
	}, E2E_FISCAL_QR);

	await page.route('**/api/v1/receipts/parse', async (route) => {
		if (route.request().method() !== 'POST') {
			await route.continue();
			return;
		}
		await route.fulfill({
			status: 200,
			contentType: 'application/json',
			body: JSON.stringify(scannedReceipt)
		});
	});
}

/** Enables receipt scanner module + API token on the server (e2e admin session). */
export async function enableReceiptScannerFeatures(page: Page) {
	await apiJSON(page, 'PUT', '/api/v1/admin/features', {
		receipt_scanner: true
	});
	await apiJSON(page, 'PUT', '/api/v1/admin/settings/receipt-api-token', {
		receipt_api_token: 'e2e-receipt-api-token'
	});
	await page.route('**/api/v1/features', async (route) => {
		if (route.request().method() !== 'GET') {
			await route.continue();
			return;
		}
		const response = await route.fetch();
		const body = (await response.json()) as Record<string, boolean>;
		await route.fulfill({
			status: response.status(),
			contentType: 'application/json',
			body: JSON.stringify({
				...body,
				receipt_scanner: true,
				receipt_api_token_set: true
			})
		});
	});
}
