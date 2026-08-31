import { describe, expect, it, vi, beforeEach } from 'vitest';
import {
	compareVersions,
	fetchAppVersionInfo,
	isBlockingVersionMismatch,
	normalizeVersion,
	releaseUrlForVersion,
	resetVersionCheckForTests,
	versionBehindBlocks,
	versionsMismatch
} from '$lib/version-check';
import { getVersionCheck } from '$lib/api/client';

vi.mock('$lib/api/client', () => ({
	getVersionCheck: vi.fn()
}));

const mockedGetVersionCheck = vi.mocked(getVersionCheck);

describe('compareVersions', () => {
	it('compares semver parts', () => {
		expect(compareVersions('1.4.0', '1.4.1')).toBe(-1);
		expect(compareVersions('1.4.0', '1.4.0')).toBe(0);
		expect(compareVersions('1.4.1', '1.4.0')).toBe(1);
		expect(compareVersions('1.3.9', '1.4.0')).toBe(-1);
	});

	it('strips v prefix', () => {
		expect(compareVersions('v1.4.0', '1.4.1')).toBe(-1);
	});
});

describe('versionBehindBlocks', () => {
	it('blocks on major or minor behind', () => {
		expect(versionBehindBlocks('1.4.0', '2.0.0')).toBe(true);
		expect(versionBehindBlocks('1.4.0', '1.5.0')).toBe(true);
	});

	it('does not block on patch-only behind', () => {
		expect(versionBehindBlocks('1.4.0', '1.4.1')).toBe(false);
		expect(versionBehindBlocks('1.4.2', '1.4.9')).toBe(false);
	});

	it('does not block when app is same or ahead', () => {
		expect(versionBehindBlocks('1.4.1', '1.4.0')).toBe(false);
		expect(versionBehindBlocks('1.5.0', '1.4.9')).toBe(false);
	});
});

describe('isBlockingVersionMismatch', () => {
	it('blocks only when versionBlocked is true', () => {
		expect(
			isBlockingVersionMismatch({
				appVersion: '1.4.0',
				serverVersion: '1.5.0',
				releaseUrl: null,
				versionMismatch: true,
				versionBlocked: true
			})
		).toBe(true);
		expect(
			isBlockingVersionMismatch({
				appVersion: '1.4.0',
				serverVersion: '1.4.1',
				releaseUrl: null,
				versionMismatch: true,
				versionBlocked: false
			})
		).toBe(false);
	});
});

describe('versionsMismatch', () => {
	it('is true only when app is older than server', () => {
		expect(versionsMismatch('1.4.0', '1.4.1')).toBe(true);
		expect(versionsMismatch('1.4.0', '1.4.0')).toBe(false);
		expect(versionsMismatch('1.4.1', '1.4.0')).toBe(false);
	});
});

describe('releaseUrlForVersion', () => {
	it('builds GitHub tag URL', () => {
		expect(releaseUrlForVersion('1.4.1')).toBe(
			'https://github.com/kai-zer-ru/buhgalter/releases/tag/v1.4.1'
		);
	});
});

describe('normalizeVersion', () => {
	it('trims and removes v prefix', () => {
		expect(normalizeVersion(' v1.4.0 ')).toBe('1.4.0');
	});
});

describe('fetchAppVersionInfo', () => {
	beforeEach(() => {
		mockedGetVersionCheck.mockReset();
		resetVersionCheckForTests();
	});

	it('flags mismatch but not block for patch behind', async () => {
		mockedGetVersionCheck.mockResolvedValue({
			current_version: '1.4.1',
			update_available: false
		});
		const result = await fetchAppVersionInfo('1.4.0');
		expect(result).toEqual({
			appVersion: '1.4.0',
			serverVersion: '1.4.1',
			releaseUrl: 'https://github.com/kai-zer-ru/buhgalter/releases/tag/v1.4.1',
			versionMismatch: true,
			versionBlocked: false
		});
	});

	it('flags block for minor behind', async () => {
		mockedGetVersionCheck.mockResolvedValue({
			current_version: '1.5.0',
			update_available: false
		});
		const result = await fetchAppVersionInfo('1.4.0');
		expect(result.versionMismatch).toBe(true);
		expect(result.versionBlocked).toBe(true);
	});

	it('returns offline shape on network error', async () => {
		mockedGetVersionCheck.mockRejectedValue(new Error('offline'));
		expect(await fetchAppVersionInfo('1.4.0')).toEqual({
			appVersion: '1.4.0',
			serverVersion: null,
			releaseUrl: null,
			versionMismatch: false,
			versionBlocked: false
		});
	});

	it('skips network within 24h and returns cached result', async () => {
		mockedGetVersionCheck.mockResolvedValue({
			current_version: '1.5.0',
			update_available: false
		});
		await fetchAppVersionInfo('1.4.0');
		mockedGetVersionCheck.mockClear();
		const second = await fetchAppVersionInfo('1.4.0');
		expect(mockedGetVersionCheck).not.toHaveBeenCalled();
		expect(second.serverVersion).toBe('1.5.0');
	});

	it('force bypasses daily throttle', async () => {
		mockedGetVersionCheck.mockResolvedValue({
			current_version: '1.4.0',
			update_available: false
		});
		await fetchAppVersionInfo('1.4.0');
		mockedGetVersionCheck.mockResolvedValue({
			current_version: '1.6.0',
			update_available: false
		});
		const forced = await fetchAppVersionInfo('1.4.0', { force: true });
		expect(mockedGetVersionCheck).toHaveBeenCalledTimes(2);
		expect(forced.serverVersion).toBe('1.6.0');
	});
});
