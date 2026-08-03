import { notifyServerDataChanged, scheduleSyncOutbox } from '$lib/offline/sync';

/** Online write succeeded — cache cleared in client.ts; reload open pages + drain outbox. */
export function afterOnlineWrite() {
	notifyServerDataChanged();
	scheduleSyncOutbox();
}
