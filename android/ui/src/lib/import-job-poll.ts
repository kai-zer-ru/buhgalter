/** Keep polling a background import job on lock/timeouts; drop it only on auth/missing job. */

export const ACTIVE_IMPORT_JOB_ID_KEY = 'buhgalter.active_import_job_id';
export const ACTIVE_IMPORT_JOB_SNAPSHOT_KEY = 'buhgalter.active_import_job';

export type ImportJobStatus = 'queued' | 'running' | 'done' | 'failed';

export type StoredImportJobSnapshot = {
	id: string;
	filename: string;
	status: ImportJobStatus;
	created_at: string;
	started_at?: string;
	error_message?: string;
	report?: {
		total_rows: number;
		processed_rows: number;
		valid_rows?: number;
		skipped_duplicates: number;
		created_transactions?: number;
		errors?: { row: number; message: string }[];
		logs?: string[];
	};
};

export function isRetriableImportJobPollError(err: unknown): boolean {
	if (!err || typeof err !== 'object') {
		return true;
	}
	const status = 'status' in err ? Number((err as { status: unknown }).status) : NaN;
	if (!Number.isFinite(status) || status === 0) {
		return true;
	}
	if (status === 401 || status === 403 || status === 404) {
		return false;
	}
	return status >= 500 || status === 408 || status === 429;
}

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
		// quota / private mode
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

export function readStoredImportJobID(): string | null {
	const id = storageGet(ACTIVE_IMPORT_JOB_ID_KEY)?.trim();
	return id || null;
}

function parseStatus(s: unknown): ImportJobStatus {
	if (s === 'queued' || s === 'running' || s === 'done' || s === 'failed') {
		return s;
	}
	return 'running';
}

export function readStoredImportJobSnapshot(): StoredImportJobSnapshot | null {
	const id = readStoredImportJobID();
	if (!id) return null;
	const raw = storageGet(ACTIVE_IMPORT_JOB_SNAPSHOT_KEY);
	if (raw) {
		try {
			const parsed = JSON.parse(raw) as StoredImportJobSnapshot;
			if (parsed && typeof parsed === 'object' && parsed.id === id) {
				return {
					id,
					filename: typeof parsed.filename === 'string' ? parsed.filename : '',
					status: parseStatus(parsed.status),
					created_at: typeof parsed.created_at === 'string' ? parsed.created_at : '',
					started_at: typeof parsed.started_at === 'string' ? parsed.started_at : undefined,
					error_message:
						typeof parsed.error_message === 'string' ? parsed.error_message : undefined,
					report: parsed.report
				};
			}
		} catch {
			// fall through to id-only placeholder
		}
	}
	return { id, filename: '', status: 'running', created_at: '' };
}

export function writeStoredImportJob(job: {
	id: string;
	filename?: string;
	status?: string;
	created_at?: string;
	started_at?: string;
	error_message?: string;
	report?: {
		total_rows?: number;
		processed_rows?: number;
		valid_rows?: number;
		skipped_duplicates?: number;
		created_transactions?: number;
		errors?: { row: number; message: string }[];
		logs?: string[];
	};
}): void {
	storageSet(ACTIVE_IMPORT_JOB_ID_KEY, job.id);
	storageSet(
		ACTIVE_IMPORT_JOB_SNAPSHOT_KEY,
		JSON.stringify({
			id: job.id,
			filename: job.filename ?? '',
			status: parseStatus(job.status),
			created_at: job.created_at ?? '',
			started_at: job.started_at,
			error_message: job.error_message,
			report: job.report
				? {
						total_rows: job.report.total_rows ?? 0,
						processed_rows: job.report.processed_rows ?? 0,
						valid_rows: job.report.valid_rows,
						skipped_duplicates: job.report.skipped_duplicates ?? 0,
						created_transactions: job.report.created_transactions,
						errors: job.report.errors,
						logs: job.report.logs
					}
				: undefined
		})
	);
}

export function clearStoredImportJob(): void {
	storageRemove(ACTIVE_IMPORT_JOB_ID_KEY);
	storageRemove(ACTIVE_IMPORT_JOB_SNAPSHOT_KEY);
}

export function restoreStepFromSnapshot(
	snapshot: StoredImportJobSnapshot | null
): 'upload' | 'importing' {
	return snapshot ? 'importing' : 'upload';
}
