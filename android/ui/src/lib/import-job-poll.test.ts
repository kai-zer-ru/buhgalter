import { afterEach, describe, expect, it } from 'vitest';
import {
	ACTIVE_IMPORT_JOB_ID_KEY,
	ACTIVE_IMPORT_JOB_SNAPSHOT_KEY,
	clearStoredImportJob,
	isRetriableImportJobPollError,
	readStoredImportJobSnapshot,
	restoreStepFromSnapshot,
	writeStoredImportJob
} from './import-job-poll';

describe('isRetriableImportJobPollError', () => {
	it('retries network failures and SQLite lock / 5xx', () => {
		expect(isRetriableImportJobPollError(new TypeError('Failed to fetch'))).toBe(true);
		expect(isRetriableImportJobPollError({ status: 500 })).toBe(true);
		expect(isRetriableImportJobPollError({ status: 503 })).toBe(true);
	});

	it('stops on missing job or session', () => {
		expect(isRetriableImportJobPollError({ status: 404 })).toBe(false);
		expect(isRetriableImportJobPollError({ status: 401 })).toBe(false);
	});
});

describe('stored import job snapshot', () => {
	const mem = new Map<string, string>();

	afterEach(() => {
		mem.clear();
	});

	function stubStorage() {
		const storage = {
			getItem: (key: string) => mem.get(key) ?? null,
			setItem: (key: string, value: string) => {
				mem.set(key, value);
			},
			removeItem: (key: string) => {
				mem.delete(key);
			}
		};
		Object.defineProperty(globalThis, 'localStorage', {
			configurable: true,
			value: storage
		});
	}

	it('shows importing immediately from a stored running snapshot', () => {
		stubStorage();
		writeStoredImportJob({
			id: 'job-1',
			filename: 'ledger.csv',
			status: 'running',
			created_at: '2026-09-18T00:00:00Z',
			started_at: '2026-09-18T00:00:01Z',
			report: { total_rows: 1000, processed_rows: 400, skipped_duplicates: 0 }
		});
		const snap = readStoredImportJobSnapshot();
		expect(restoreStepFromSnapshot(snap)).toBe('importing');
		expect(snap?.filename).toBe('ledger.csv');
		expect(snap?.report?.processed_rows).toBe(400);
		expect(mem.get(ACTIVE_IMPORT_JOB_ID_KEY)).toBe('job-1');
		expect(mem.get(ACTIVE_IMPORT_JOB_SNAPSHOT_KEY)).toContain('ledger.csv');
	});

	it('falls back to a running placeholder when only the job id is stored', () => {
		stubStorage();
		mem.set(ACTIVE_IMPORT_JOB_ID_KEY, 'job-legacy');
		const snap = readStoredImportJobSnapshot();
		expect(snap).toEqual({
			id: 'job-legacy',
			filename: '',
			status: 'running',
			created_at: ''
		});
		expect(restoreStepFromSnapshot(snap)).toBe('importing');
	});

	it('clears both keys', () => {
		stubStorage();
		writeStoredImportJob({
			id: 'job-1',
			filename: 'a.csv',
			status: 'running',
			created_at: ''
		});
		clearStoredImportJob();
		expect(readStoredImportJobSnapshot()).toBeNull();
		expect(restoreStepFromSnapshot(null)).toBe('upload');
	});
});
