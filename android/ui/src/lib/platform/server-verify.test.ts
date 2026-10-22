import { describe, expect, it } from 'vitest';
import { addTrustedOrigin } from './server-verify';
import { normalizeProfile } from './server-profile';

describe('server-verify trust', () => {
	it('adds https origin to trusted list', () => {
		const profile = normalizeProfile({
			lanUrl: 'http://192.168.1.1:8765',
			remoteUrl: 'https://buh.example.com',
			homeSsids: [],
			trustedOrigins: []
		});
		const next = addTrustedOrigin(profile, 'https://buh.example.com', true);
		expect(next.trustedOrigins).toEqual(['https://buh.example.com']);
	});

	it('ignores http origins for trust', () => {
		const profile = normalizeProfile({ lanUrl: 'http://192.168.1.1:8765', homeSsids: [] });
		const next = addTrustedOrigin(profile, 'http://192.168.1.1:8765', true);
		expect(next.trustedOrigins).toEqual([]);
	});
});
