import { describe, expect, it, beforeEach } from 'vitest';
import { normalizeServerUrl } from './server-origin';
import {
	getServerUrl,
	hasServerUrl,
	resetServerUrlForTests,
	refreshActiveServerUrl,
	setLastKnownSsidForTests,
	setServerUrl
} from './server-url';
import { setServerProfile } from './server-profile';

describe('server-url', () => {
	beforeEach(() => {
		resetServerUrlForTests();
	});

	it('normalizes user input to origin', () => {
		expect(normalizeServerUrl('https://buh.example.com/path')).toBe('https://buh.example.com');
		expect(normalizeServerUrl('buh.example.com')).toBe('https://buh.example.com');
	});

	it('treats empty profile as unconfigured', () => {
		expect(hasServerUrl()).toBe(false);
		expect(getServerUrl()).toBe('');
	});

	it('resolves LAN on home SSID when remote is configured', async () => {
		setServerProfile({
			lanUrl: 'http://192.168.1.10:8765',
			remoteUrl: 'https://buh.example.com',
			homeSsids: ['HomeWiFi']
		});
		setLastKnownSsidForTests('HomeWiFi');
		await refreshActiveServerUrl();
		expect(getServerUrl()).toBe('http://192.168.1.10:8765');
	});

	it('resolves remote off home Wi‑Fi', async () => {
		setServerProfile({
			lanUrl: 'http://192.168.1.10:8765',
			remoteUrl: 'https://buh.example.com',
			homeSsids: ['HomeWiFi']
		});
		setLastKnownSsidForTests('Cafe');
		await refreshActiveServerUrl();
		expect(getServerUrl()).toBe('https://buh.example.com');
	});

	it('setServerUrl stores LAN URL in profile', () => {
		setServerUrl('http://192.168.1.10:8765');
		expect(getServerUrl()).toBe('http://192.168.1.10:8765');
	});
});
