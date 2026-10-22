import { beforeEach, describe, expect, it, vi } from 'vitest';
import { buildOutboxExportPayload, exportOutboxForSupport } from '$lib/offline/export';
import { resetOutboxForTests } from '$lib/offline/store';

vi.mock('$lib/platform/app-version', () => ({
	APP_VERSION: '1.4.0'
}));

vi.mock('$lib/platform/server-url', () => ({
	getServerUrl: () => 'http://192.168.1.10:8765'
}));

describe('exportOutboxForSupport', () => {
	beforeEach(() => {
		resetOutboxForTests();
	});

	it('returns JSON with metadata and entries', () => {
		const payload = buildOutboxExportPayload();
		expect(payload.appVersion).toBe('1.4.0');
		expect(payload.serverUrl).toBe('http://192.168.1.10:8765');
		expect(payload.entries).toEqual([]);
		expect(exportOutboxForSupport()).toContain('"appVersion": "1.4.0"');
	});
});
