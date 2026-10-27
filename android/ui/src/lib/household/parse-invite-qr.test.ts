import { describe, expect, it } from 'vitest';
import { parseInviteQr } from './parse-invite-qr';

const origin = 'https://home.example:8765';

describe('parseInviteQr', () => {
	it('accepts a matching http(s) invite URL', () => {
		expect(parseInviteQr('https://home.example:8765/invite/ABC-DEF-GHJ', origin)).toEqual({
			ok: true,
			code: 'ABCDEFGHJ'
		});
		expect(parseInviteQr('http://home.example:8765/invite/ABCDEFGHJ', origin)).toEqual({
			ok: true,
			code: 'ABCDEFGHJ'
		});
	});

	it('accepts the app scheme', () => {
		expect(parseInviteQr('ru.kai_zer.buhgalter://invite/ABC-DEF-GHJ', origin)).toEqual({
			ok: true,
			code: 'ABCDEFGHJ'
		});
	});

	it('accepts a bare code', () => {
		expect(parseInviteQr('abc-def-ghj', origin)).toEqual({ ok: true, code: 'ABCDEFGHJ' });
	});

	it('warns on a foreign host and does not return the code as ok', () => {
		expect(parseInviteQr('https://other.example/invite/ABC-DEF-GHJ', origin)).toEqual({
			ok: false,
			reason: 'foreign_host',
			host: 'other.example'
		});
	});

	it('accepts a LAN-only QR whose host is not the saved IP', () => {
		expect(
			parseInviteQr('http://buhgalter.local:8766/invite/WU4-UNG-A56', 'http://192.168.1.28:8766', {
				lanUrl: 'http://192.168.1.28:8766',
				remoteUrl: ''
			})
		).toEqual({ ok: true, code: 'WU4UNGA56' });
	});

	it('accepts the remote URL of the same instance while LAN is active', () => {
		expect(
			parseInviteQr(
				'https://bbbbbb1111bbbbbb.kai-zer.ru/invite/WU4-UNG-A56',
				'http://192.168.1.28:8766',
				{
					lanUrl: 'http://192.168.1.28:8766',
					remoteUrl: 'https://bbbbbb1111bbbbbb.kai-zer.ru'
				}
			)
		).toEqual({ ok: true, code: 'WU4UNGA56' });
	});

	it('rejects garbage', () => {
		expect(parseInviteQr('https://home.example:8765/accounts', origin)).toEqual({
			ok: false,
			reason: 'invalid'
		});
		expect(parseInviteQr('not-a-code', origin)).toEqual({ ok: false, reason: 'invalid' });
	});
});
