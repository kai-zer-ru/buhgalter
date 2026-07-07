const ACTIVE_RECEIPT_SCAN_JOB_KEY = 'buhgalter.active_receipt_scan_job_id';

export function saveActiveReceiptScanJobId(jobId: string) {
	sessionStorage.setItem(ACTIVE_RECEIPT_SCAN_JOB_KEY, jobId);
}

export function getActiveReceiptScanJobId(): string | null {
	return sessionStorage.getItem(ACTIVE_RECEIPT_SCAN_JOB_KEY);
}

export function clearActiveReceiptScanJobId() {
	sessionStorage.removeItem(ACTIVE_RECEIPT_SCAN_JOB_KEY);
}
