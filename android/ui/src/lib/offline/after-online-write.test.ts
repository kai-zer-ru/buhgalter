import { beforeEach, describe, expect, it, vi } from 'vitest';
import { get } from 'svelte/store';
import { afterOnlineWrite } from '$lib/offline/after-online-write';
import { dataRefreshTick } from '$lib/offline/sync';
import * as sync from '$lib/offline/sync';

vi.mock('$lib/offline/sync', async (importOriginal) => {
	const actual = await importOriginal<typeof sync>();
	return {
		...actual,
		scheduleSyncOutbox: vi.fn()
	};
});

beforeEach(() => {
	dataRefreshTick.set(0);
	vi.clearAllMocks();
});

describe('afterOnlineWrite', () => {
	it('bumps dataRefreshTick and schedules outbox sync', () => {
		afterOnlineWrite();
		expect(get(dataRefreshTick)).toBe(1);
		expect(sync.scheduleSyncOutbox).toHaveBeenCalledOnce();
	});
});
