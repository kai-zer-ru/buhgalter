import { get } from 'svelte/store';
import { APP_VERSION } from '$lib/platform/app-version';
import { isNativeApp } from '$lib/platform/native';
import { getServerProfile } from '$lib/platform/server-profile';
import { getServerUrl } from '$lib/platform/server-url';
import { serverReachability } from '$lib/offline/server-connectivity';
import { pendingOutboxCount, failedOutboxCount } from '$lib/offline/store';
import { saveDebugLogFile } from '$lib/platform/debug-export';

const ENABLED_KEY = 'buhgalter.debug_log.enabled';
const ENTRIES_KEY = 'buhgalter.debug_log.entries';
const MAX_ENTRIES = 3000;

export type DebugLogLevel = 'debug' | 'info' | 'warn' | 'error';

export type DebugLogEntry = {
	ts: string;
	level: DebugLogLevel;
	category: string;
	message: string;
	data?: Record<string, unknown>;
};

let memoryEnabled: boolean | null = null;
let memoryEntries: DebugLogEntry[] | null = null;

function storageGet(key: string): string | null {
	if (typeof localStorage === 'undefined') return null;
	try {
		return localStorage.getItem(key);
	} catch {
		return null;
	}
}

function storageSet(key: string, value: string): void {
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.setItem(key, value);
	} catch {
		// quota
	}
}

function storageRemove(key: string): void {
	if (typeof localStorage === 'undefined') return;
	try {
		localStorage.removeItem(key);
	} catch {
		// ignore
	}
}

export function isDebugLogEnabled(): boolean {
	if (memoryEnabled !== null) return memoryEnabled;
	const raw = storageGet(ENABLED_KEY);
	memoryEnabled = raw === '1';
	return memoryEnabled;
}

function persistEntries(entries: DebugLogEntry[]): void {
	memoryEntries = entries;
	try {
		storageSet(ENTRIES_KEY, JSON.stringify(entries));
	} catch {
		// ignore
	}
}

function loadEntries(): DebugLogEntry[] {
	if (memoryEntries) return memoryEntries;
	const raw = storageGet(ENTRIES_KEY);
	if (!raw) {
		memoryEntries = [];
		return memoryEntries;
	}
	try {
		memoryEntries = JSON.parse(raw) as DebugLogEntry[];
		return memoryEntries;
	} catch {
		memoryEntries = [];
		return memoryEntries;
	}
}

export function clearDebugLogEntries(): void {
	memoryEntries = [];
	storageRemove(ENTRIES_KEY);
}

export function getDebugLogEntries(): DebugLogEntry[] {
	return [...loadEntries()];
}

function appendEntry(entry: DebugLogEntry): void {
	if (!isDebugLogEnabled()) return;
	const entries = loadEntries();
	entries.push(entry);
	while (entries.length > MAX_ENTRIES) entries.shift();
	persistEntries(entries);
}

export function debugLog(
	level: DebugLogLevel,
	category: string,
	message: string,
	data?: Record<string, unknown>
): void {
	if (!isDebugLogEnabled()) return;
	appendEntry({
		ts: new Date().toISOString(),
		level,
		category,
		message,
		data: data ? redactData(data) : undefined
	});
}

export function debugLogInfo(
	category: string,
	message: string,
	data?: Record<string, unknown>
): void {
	debugLog('info', category, message, data);
}

export function debugLogWarn(
	category: string,
	message: string,
	data?: Record<string, unknown>
): void {
	debugLog('warn', category, message, data);
}

export function debugLogError(
	category: string,
	message: string,
	data?: Record<string, unknown>
): void {
	debugLog('error', category, message, data);
}

const SENSITIVE_KEYS = /^(authorization|token|password|pin|secret|api[_-]?token)$/i;

export function redactHeaders(headers: Record<string, string>): Record<string, string> {
	const out: Record<string, string> = {};
	for (const [key, value] of Object.entries(headers)) {
		if (SENSITIVE_KEYS.test(key) || key.toLowerCase() === 'authorization') {
			out[key] = redactSecret(value);
		} else {
			out[key] = value;
		}
	}
	return out;
}

function redactSecret(value: string): string {
	const trimmed = value.trim();
	if (/^bearer\s+/i.test(trimmed)) return 'Bearer ***';
	if (trimmed.length <= 4) return '***';
	return `${trimmed.slice(0, 2)}***${trimmed.slice(-2)}`;
}

function redactData(data: Record<string, unknown>): Record<string, unknown> {
	const out: Record<string, unknown> = {};
	for (const [key, value] of Object.entries(data)) {
		if (SENSITIVE_KEYS.test(key)) {
			out[key] = typeof value === 'string' ? redactSecret(value) : '***';
			continue;
		}
		if (key === 'headers' && value && typeof value === 'object') {
			out[key] = redactHeaders(value as Record<string, string>);
			continue;
		}
		if (typeof value === 'string' && value.length > 2000) {
			out[key] = `${value.slice(0, 2000)}… [truncated ${value.length} chars]`;
			continue;
		}
		out[key] = value;
	}
	return out;
}

export function serializeError(err: unknown): Record<string, unknown> {
	if (err && typeof err === 'object' && 'code' in err && 'status' in err) {
		const api = err as { code?: string; message?: string; status?: number; field?: string };
		return {
			type: 'ApiError',
			code: api.code,
			message: api.message,
			status: api.status,
			field: api.field
		};
	}
	if (err instanceof Error) {
		return { type: err.name, message: err.message, stack: err.stack };
	}
	return { type: 'unknown', value: String(err) };
}

function environmentSnapshot(): Record<string, unknown> {
	return {
		appVersion: APP_VERSION,
		native: isNativeApp(),
		userAgent: typeof navigator !== 'undefined' ? navigator.userAgent : '',
		language: typeof navigator !== 'undefined' ? navigator.language : '',
		serverUrl: getServerUrl(),
		serverProfile: getServerProfile(),
		serverReachability: get(serverReachability),
		outboxPending: pendingOutboxCount(),
		outboxFailed: failedOutboxCount(),
		online: typeof navigator !== 'undefined' ? navigator.onLine : null
	};
}

export function buildDebugLogExport(): string {
	const lines: string[] = [
		'=== Buhgalter debug log ===',
		`Exported: ${new Date().toISOString()}`,
		'',
		'--- Environment ---',
		JSON.stringify(environmentSnapshot(), null, 2),
		'',
		'--- Events ---'
	];
	for (const e of getDebugLogEntries()) {
		const data = e.data ? ` ${JSON.stringify(e.data)}` : '';
		lines.push(`${e.ts} [${e.level}] [${e.category}] ${e.message}${data}`);
	}
	lines.push('');
	lines.push(`--- End (${getDebugLogEntries().length} events) ---`);
	return lines.join('\n');
}

export function exportFilename(): string {
	const stamp = new Date().toISOString().replace(/[:.]/g, '-').slice(0, 19);
	return `buhgalter-debug-${stamp}.log`;
}

export async function exportDebugLogToDownloads(): Promise<string> {
	const content = buildDebugLogExport();
	const filename = exportFilename();
	return saveDebugLogFile(filename, content);
}

export function setDebugLogEnabled(enabled: boolean): void {
	memoryEnabled = enabled;
	storageSet(ENABLED_KEY, enabled ? '1' : '0');
	if (enabled) {
		clearDebugLogEntries();
		debugLogInfo('session', 'Debug logging started', environmentSnapshot());
	}
}

let listenersInstalled = false;
let prevReachability: import('$lib/offline/server-connectivity').ServerReachability | undefined;

export function initDebugLogListeners(): void {
	if (listenersInstalled || typeof window === 'undefined') return;
	listenersInstalled = true;

	window.addEventListener('error', (event) => {
		debugLogError('uncaught', event.message, {
			filename: event.filename,
			lineno: event.lineno,
			colno: event.colno,
			stack: event.error instanceof Error ? event.error.stack : undefined
		});
	});

	window.addEventListener('unhandledrejection', (event) => {
		debugLogError('unhandledrejection', 'Unhandled promise rejection', {
			reason: serializeError(event.reason)
		});
	});

	serverReachability.subscribe((reachability) => {
		if (prevReachability !== undefined && reachability !== prevReachability) {
			debugLogInfo('connectivity', `Server reachability: ${prevReachability} → ${reachability}`);
		}
		prevReachability = reachability;
	});
}

export function logApiRequest(
	base: string,
	path: string,
	method: string,
	headers: Record<string, string>,
	body?: unknown
): number {
	if (!isDebugLogEnabled()) return 0;
	const bodyText =
		typeof body === 'string' ? (body.length > 2000 ? `${body.slice(0, 2000)}…` : body) : body;
	debugLog('debug', 'api', `→ ${method} ${path}`, {
		url: `${base}${path}`,
		headers: redactHeaders(headers),
		body: bodyText
	});
	return Date.now();
}

export function logApiResponse(
	path: string,
	method: string,
	startedAt: number,
	info: { status?: number; ok?: boolean; error?: unknown; fromCache?: boolean }
): void {
	if (!isDebugLogEnabled() || !startedAt) return;
	const ms = Date.now() - startedAt;
	if (info.error) {
		debugLogError('api', `✗ ${method} ${path} (${ms}ms)`, {
			status: info.status,
			...serializeError(info.error)
		});
		return;
	}
	const cacheTag = info.fromCache ? ' [cache]' : '';
	debugLogInfo('api', `← ${method} ${path} ${info.status ?? 200} (${ms}ms)${cacheTag}`);
}

export function resetDebugLogForTests(): void {
	memoryEnabled = null;
	memoryEntries = null;
	storageRemove(ENABLED_KEY);
	storageRemove(ENTRIES_KEY);
	listenersInstalled = false;
}
