import { describe, expect, it, beforeEach } from 'vitest';
import {
	isConnectionError,
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

describe('serverReachability', () => {
	it('markServerOffline sets offline state', () => {
		markServerOnline();
		markServerOffline();
		expect(get(serverReachability)).toBe('offline');
	});
});
