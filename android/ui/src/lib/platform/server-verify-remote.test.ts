import { beforeEach, describe, expect, it, vi } from 'vitest';

const pingServer = vi.hoisted(() => vi.fn());

vi.mock('$lib/api/client', async (importOriginal) => {
	const actual = await importOriginal<typeof import('$lib/api/client')>();
	return { ...actual, pingServer };
});

import { ApiError } from '$lib/api/client';
import { verifyServerProfile } from './server-verify';

const LAN = 'http://192.168.1.10:8765';
const REMOTE = 'https://buh.example.com';

describe('verifyServerProfile remote from discovery', () => {
	beforeEach(() => {
		pingServer.mockReset();
		pingServer.mockImplementation(async (origin: string) => {
			if (origin === REMOTE) throw new ApiError('UNREACHABLE', 'no hairpin', 0);
		});
	});

	it('fails on an unreachable remote URL by default', async () => {
		await expect(verifyServerProfile({ lanUrl: LAN, remoteUrl: REMOTE })).rejects.toThrow();
	});

	it('saves with an unreachable remote URL when it came from discovery', async () => {
		const onRemoteUnreachable = vi.fn();
		const profile = await verifyServerProfile(
			{ lanUrl: LAN, remoteUrl: REMOTE },
			{ onRemoteUnreachable }
		);
		expect(profile.remoteUrl).toBe(REMOTE);
		expect(onRemoteUnreachable).toHaveBeenCalledOnce();
	});

	it('still fails when the LAN URL is unreachable', async () => {
		pingServer.mockRejectedValue(new ApiError('UNREACHABLE', 'down', 0));
		await expect(
			verifyServerProfile({ lanUrl: LAN, remoteUrl: REMOTE }, { onRemoteUnreachable: vi.fn() })
		).rejects.toThrow();
	});
});
