import { writable } from 'svelte/store';

/** Bump to reload pending-user banners after moderation. */
export const pendingUsersBannerTick = writable(0);

export function refreshPendingUsersBanner(): void {
	pendingUsersBannerTick.update((n) => n + 1);
}
