import { describe, expect, it, beforeEach } from 'vitest';
import {
	isConnectionError,
	isServerUnavailableForUser,
	markServerOffline,
	markServerOnline,
	serverReachability,
	stopServerProbeLoopForTests
} from '$lib/offline/server-connectivity';
import { get } from 'svelte/store';

beforeEach(() => {
	stopServerProbeLoopForTests();
});

describe('isConnectionError', () => {
	it('detects Capacitor failed to connect message', () => {
		expect(isConnectionError(new Error('Failed to connect to /192.168.0.10:8766'))).toBe(true);
	});

	it('detects TypeError', () => {
		expect(isConnectionError(new TypeError('Failed to fetch'))).toBe(true);
	});

	it('ignores validation errors', () => {
		expect(isConnectionError(new Error('invalid amount'))).toBe(false);
	});
});

describe('isServerUnavailableForUser', () => {
	it('covers connect failures', () => {
		expect(isServerUnavailableForUser(new Error('Failed to connect to /192.168.0.10:8766'))).toBe(
			true
		);
	});

	it('covers UNREACHABLE API errors', () => {
		expect(isServerUnavailableForUser({ code: 'UNREACHABLE', message: 'Could not connect' })).toBe(
			true
		);
	});

	it('covers offline cache miss without treating it as a generic Error', () => {
		const err = new Error('No cached data for /api/v1/transactions?page=2');
		err.name = 'OfflineCacheMissError';
		expect(isServerUnavailableForUser(err)).toBe(true);
		expect(isConnectionError(err)).toBe(false);
	});

	it('ignores validation errors', () => {
		expect(isServerUnavailableForUser(new Error('invalid amount'))).toBe(false);
	});

	it('does not treat SSL_CERTIFICATE as a generic outage', () => {
		expect(
			isServerUnavailableForUser({
				code: 'SSL_CERTIFICATE',
				status: 0,
				message: 'Untrusted certificate'
			})
		).toBe(false);
	});
});

describe('serverReachability', () => {
	it('markServerOffline sets offline state', () => {
		markServerOnline();
		markServerOffline();
		expect(get(serverReachability)).toBe('offline');
	});

	it('markServerOnline is a no-op when already online', () => {
		markServerOnline();
		let notifies = 0;
		const unsub = serverReachability.subscribe(() => {
			notifies++;
		});
		notifies = 0;
		markServerOnline();
		markServerOnline();
		expect(notifies).toBe(0);
		unsub();
	});
});
