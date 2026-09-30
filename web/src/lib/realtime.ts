/** Live cache-invalidation channel for the web UI (WebSocket). Android stays on SWR poll. */

import { get, writable } from 'svelte/store';
import type { ImportJob, ImportReport } from '$lib/api/client';
import {
	LEDGER_BALANCE_HINT_PATHS,
	notifyRealtimeInvalidate,
	notifyRealtimeInvalidatePaths,
	setRealtimeLive,
	wasRecentLocalMutation
} from '$lib/ref-cache';
import { catchUpAfterReconnect, seedTransactionChangesCursor } from '$lib/tx-changes-catchup';

export type RealtimeStatus = 'idle' | 'connecting' | 'open' | 'closed';

export type ImportRealtimeEvent = {
	type: 'import.progress' | 'import.done' | 'import.failed';
	job_id: string;
	status: ImportJob['status'];
	report?: ImportReport;
	error_message?: string;
};

/** True while the socket is open — ref-cache skips background SWR revalidate. */
export const realtimeLive = writable(false);

export const realtimeStatus = writable<RealtimeStatus>('idle');

/** Last import.* event from the socket (optional store for debugging / peek). */
export const importRealtimeEvent = writable<ImportRealtimeEvent | null>(null);

/** Buffer terminal/progress events by job id — job can finish before createImportJob returns. */
const importEventsByJob = new Map<string, ImportRealtimeEvent>();

/** Sync listeners — must run inside onmessage before the next WS frame (invalidate). */
type ImportRealtimeListener = (ev: ImportRealtimeEvent) => void;
const importListeners = new Set<ImportRealtimeListener>();

const REALTIME_PATH = '/api/v1/realtime';

let socket: WebSocket | null = null;
let wantConnected = false;
let reconnectAttempt = 0;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let visibilityBound = false;
/** True after at least one successful open — next open is a reconnect. */
let everOpened = false;

function wsURL(): string {
	const proto = location.protocol === 'https:' ? 'wss:' : 'ws:';
	return `${proto}//${location.host}${REALTIME_PATH}`;
}

function clearReconnectTimer(): void {
	if (reconnectTimer !== null) {
		clearTimeout(reconnectTimer);
		reconnectTimer = null;
	}
}

function scheduleReconnect(): void {
	clearReconnectTimer();
	if (!wantConnected) return;
	const delay = Math.min(30_000, 1000 * 2 ** Math.min(reconnectAttempt, 5));
	reconnectAttempt += 1;
	reconnectTimer = setTimeout(() => {
		reconnectTimer = null;
		connectRealtime();
	}, delay);
}

/**
 * Subscribe to import.* events synchronously from the WebSocket onmessage path.
 * ImportTab must apply `import.done` here (not via `$effect`) so it runs before
 * the following `invalidate` frame and can markLocalMutation to ignore that echo.
 */
export function subscribeImportRealtime(listener: ImportRealtimeListener): () => void {
	importListeners.add(listener);
	return () => {
		importListeners.delete(listener);
	};
}

function notifyImportListeners(ev: ImportRealtimeEvent): void {
	for (const listener of importListeners) {
		try {
			listener(ev);
		} catch {
			// one bad subscriber must not break the socket loop
		}
	}
}

function handleImportMessage(data: {
	type?: string;
	job_id?: string;
	status?: string;
	report?: ImportReport;
	error_message?: string;
}): void {
	const type = data.type;
	if (type !== 'import.progress' && type !== 'import.done' && type !== 'import.failed') {
		return;
	}
	const jobId = data.job_id?.trim();
	if (!jobId) return;
	const status = type === 'import.done' ? 'done' : type === 'import.failed' ? 'failed' : 'running';
	const ev: ImportRealtimeEvent = {
		type,
		job_id: jobId,
		status,
		report: data.report,
		error_message: data.error_message
	};
	importEventsByJob.set(jobId, ev);
	importRealtimeEvent.set(ev);
	notifyImportListeners(ev);
}

/** Latest buffered import.* event for a job (survives race before ImportTab knows the id). */
export function peekImportRealtimeEvent(jobId: string): ImportRealtimeEvent | null {
	return importEventsByJob.get(jobId) ?? null;
}

export function clearImportRealtimeEvent(jobId: string): void {
	importEventsByJob.delete(jobId);
	const cur = get(importRealtimeEvent);
	if (cur?.job_id === jobId) {
		importRealtimeEvent.set(null);
	}
}

function handleMessage(raw: string): void {
	let data: {
		type?: string;
		job_id?: string;
		status?: string;
		report?: ImportReport;
		error_message?: string;
		hint_paths?: string[];
		entities?: string[];
	};
	try {
		data = JSON.parse(raw) as typeof data;
	} catch {
		return;
	}
	if (data.type === 'invalidate') {
		if (wasRecentLocalMutation()) return;
		const hints = (data.hint_paths ?? []).map((p) => p.trim()).filter(Boolean);
		if (hints.length === 0) {
			notifyRealtimeInvalidate();
			return;
		}
		notifyRealtimeInvalidatePaths(hints);
		return;
	}
	if (data.type?.startsWith('import.')) {
		handleImportMessage(data);
	}
}

function onVisibilityChange(): void {
	if (document.visibilityState === 'visible' && wantConnected && !isSocketOpen()) {
		reconnectAttempt = 0;
		connectRealtime();
	}
}

function isSocketOpen(): boolean {
	return socket?.readyState === WebSocket.OPEN;
}

export function isRealtimeSocketLive(): boolean {
	return get(realtimeLive);
}

export function connectRealtime(): void {
	if (typeof WebSocket === 'undefined') return;
	wantConnected = true;
	if (!visibilityBound && typeof document !== 'undefined') {
		document.addEventListener('visibilitychange', onVisibilityChange);
		visibilityBound = true;
	}
	if (
		socket &&
		(socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)
	) {
		return;
	}

	realtimeStatus.set('connecting');
	realtimeLive.set(false);
	setRealtimeLive(false);

	let ws: WebSocket;
	try {
		ws = new WebSocket(wsURL());
	} catch {
		realtimeStatus.set('closed');
		scheduleReconnect();
		return;
	}
	socket = ws;

	ws.onopen = () => {
		const isReconnect = everOpened;
		everOpened = true;
		reconnectAttempt = 0;
		realtimeStatus.set('open');
		realtimeLive.set(true);
		setRealtimeLive(true);
		if (isReconnect) {
			void catchUpAfterReconnect();
		} else {
			void seedTransactionChangesCursor();
			// Previous-session SWR can keep wrong balances while the live socket
			// disables background revalidate — drop ledger paths once on first open.
			notifyRealtimeInvalidatePaths([...LEDGER_BALANCE_HINT_PATHS]);
		}
	};

	ws.onmessage = (ev) => {
		if (typeof ev.data === 'string') handleMessage(ev.data);
	};

	ws.onerror = () => {
		// onclose follows
	};

	ws.onclose = () => {
		if (socket === ws) socket = null;
		realtimeLive.set(false);
		setRealtimeLive(false);
		realtimeStatus.set('closed');
		scheduleReconnect();
	};
}

export function disconnectRealtime(): void {
	wantConnected = false;
	clearReconnectTimer();
	reconnectAttempt = 0;
	everOpened = false;
	if (visibilityBound && typeof document !== 'undefined') {
		document.removeEventListener('visibilitychange', onVisibilityChange);
		visibilityBound = false;
	}
	const ws = socket;
	socket = null;
	realtimeLive.set(false);
	setRealtimeLive(false);
	realtimeStatus.set('idle');
	importRealtimeEvent.set(null);
	importEventsByJob.clear();
	importListeners.clear();
	if (ws && ws.readyState < WebSocket.CLOSING) {
		ws.close();
	}
}

/** Test helper. */
export function resetRealtimeForTests(): void {
	disconnectRealtime();
}

/** Test helper: drive handleMessage without a socket. */
export function deliverRealtimeMessageForTests(raw: string): void {
	handleMessage(raw);
}
