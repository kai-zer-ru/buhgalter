import { notifyServerDataChanged, scheduleSyncOutbox } from '$lib/offline/sync';

/** Online write succeeded — snapshots kept, next GET refreshes; reload open pages + drain outbox. */
export function afterOnlineWrite() {
	notifyServerDataChanged();
	scheduleSyncOutbox();
}
