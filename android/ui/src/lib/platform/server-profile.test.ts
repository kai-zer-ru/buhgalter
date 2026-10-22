import { describe, expect, it } from 'vitest';
import {
	isHomeSsid,
	normalizeHomeSsids,
	normalizeProfile,
	resolveActiveServerUrl
} from './server-profile';

describe('normalizeHomeSsids', () => {
	it('trims, dedupes and caps at 5', () => {
		expect(normalizeHomeSsids([' Home ', 'Home', 'A', 'B', 'C', 'D', 'E'])).toEqual([
			'Home',
			'A',
			'B',
			'C',
			'D'
		]);
	});
});

describe('resolveActiveServerUrl', () => {
	const profile = {
		lanUrl: 'http://192.168.1.10:8765',
		remoteUrl: 'https://buh.example.com',
		homeSsids: ['HomeWiFi', 'Guest-2.4'],
		lanFallbackRemote: false
	};

	it('uses LAN when remote is empty', () => {
		expect(
			resolveActiveServerUrl('Other', {
				...profile,
				remoteUrl: ''
			})
		).toEqual({ url: profile.lanUrl, mode: 'lan' });
	});

	it('uses remote when only remote is set', () => {
		expect(
			resolveActiveServerUrl(null, {
				lanUrl: '',
				remoteUrl: profile.remoteUrl,
				homeSsids: []
			})
		).toEqual({ url: profile.remoteUrl, mode: 'remote' });
	});

	it('uses LAN on listed home SSID', () => {
		expect(resolveActiveServerUrl('HomeWiFi', profile)).toEqual({
			url: profile.lanUrl,
			mode: 'lan'
		});
	});

	it('uses remote on unknown SSID', () => {
		expect(resolveActiveServerUrl('CafeWiFi', profile)).toEqual({
			url: profile.remoteUrl,
			mode: 'remote'
		});
	});

	it('uses remote when SSID is unavailable', () => {
		expect(resolveActiveServerUrl(null, profile)).toEqual({
			url: profile.remoteUrl,
			mode: 'remote'
		});
	});
});

describe('normalizeProfile', () => {
	it('normalizes origins and SSIDs', () => {
		expect(
			normalizeProfile({
				lanUrl: 'http://192.168.1.10:8765/',
				remoteUrl: 'buh.example.com',
				homeSsids: [' A ', 'A']
			})
		).toEqual({
			lanUrl: 'http://192.168.1.10:8765',
			remoteUrl: 'https://buh.example.com',
			homeSsids: ['A'],
			lanFallbackRemote: false,
			trustedOrigins: []
		});
	});

	it('preserves lanFallbackRemote when set', () => {
		expect(
			normalizeProfile({
				lanUrl: 'http://192.168.1.10:8765',
				remoteUrl: 'https://buh.example.com',
				lanFallbackRemote: true
			}).lanFallbackRemote
		).toBe(true);
	});
});

describe('isHomeSsid', () => {
	it('is case-sensitive', () => {
		expect(isHomeSsid('homewifi', ['HomeWiFi'])).toBe(false);
		expect(isHomeSsid('HomeWiFi', ['HomeWiFi'])).toBe(true);
	});
});
