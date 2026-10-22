import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import {
	buildDebugLogExport,
	clearDebugLogEntries,
	debugLogInfo,
	exportFilename,
	getDebugLogEntries,
	isDebugLogEnabled,
	redactHeaders,
	resetDebugLogForTests,
	setDebugLogEnabled
} from './debug-log';

vi.mock('$lib/platform/server-url', () => ({
	getServerUrl: () => 'http://192.168.1.1:8765'
}));

vi.mock('$lib/platform/server-profile', () => ({
	getServerProfile: () => ({
		lanUrl: 'http://192.168.1.1:8765',
		remoteUrl: '',
		homeSsids: [],
		lanFallbackRemote: false,
		trustedOrigins: []
	})
}));

vi.mock('$lib/offline/server-connectivity', () => ({
	serverReachability: { subscribe: () => () => undefined }
}));

vi.mock('$lib/offline/store', () => ({
	pendingOutboxCount: () => 0,
	failedOutboxCount: () => 0
}));

vi.mock('$lib/platform/native', () => ({
	isNativeApp: () => false
}));

describe('debug-log', () => {
	beforeEach(() => {
		resetDebugLogForTests();
	});

	afterEach(() => {
		resetDebugLogForTests();
	});

	it('redacts authorization header', () => {
		expect(redactHeaders({ Authorization: 'Bearer secret-token-xyz' })).toEqual({
			Authorization: 'Bearer ***'
		});
	});

	it('records entries when enabled', () => {
		setDebugLogEnabled(true);
		expect(isDebugLogEnabled()).toBe(true);
		debugLogInfo('test', 'hello');
		expect(getDebugLogEntries().some((e) => e.message === 'hello')).toBe(true);
	});

	it('does not record when disabled', () => {
		setDebugLogEnabled(false);
		debugLogInfo('test', 'hidden');
		expect(getDebugLogEntries()).toHaveLength(0);
	});

	it('builds export with environment and events', () => {
		setDebugLogEnabled(true);
		debugLogInfo('api', 'GET /health');
		const text = buildDebugLogExport();
		expect(text).toContain('=== Buhgalter debug log ===');
		expect(text).toContain('--- Environment ---');
		expect(text).toContain('GET /health');
	});

	it('clears entries on enable', () => {
		setDebugLogEnabled(true);
		debugLogInfo('test', 'first');
		setDebugLogEnabled(true);
		expect(getDebugLogEntries().filter((e) => e.message === 'first')).toHaveLength(0);
	});

	it('exportFilename is safe', () => {
		expect(exportFilename()).toMatch(/^buhgalter-debug-.*\.log$/);
	});

	it('clearDebugLogEntries removes stored events', () => {
		setDebugLogEnabled(true);
		debugLogInfo('test', 'x');
		clearDebugLogEntries();
		expect(getDebugLogEntries()).toHaveLength(0);
	});
});
