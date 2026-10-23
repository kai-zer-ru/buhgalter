import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { ParsedReceipt, Transaction } from '$lib/api/client';
import { ApiError } from '$lib/api/client';
import { attachParsedReceiptToTransaction } from './attach-receipt';

vi.mock('$lib/api/client', async (importOriginal) => {
	const actual = await importOriginal<typeof import('$lib/api/client')>();
	return {
		...actual,
		attachReceipt: vi.fn()
	};
});

import { attachReceipt } from '$lib/api/client';

const receipt: ParsedReceipt = {
	fiscal_drive_number: '1',
	fiscal_document_number: '2',
	fiscal_sign: '3',
	date_time: '2026-01-01T12:00:00Z',
	total_sum: 5000,
	store_name: 'Shop',
	items: [],
	fns_check_url: 'https://example.com'
};

function tx(amount: number): Transaction {
	return {
		id: 'tx-1',
		account_id: 'acc-1',
		type: 'expense',
		amount,
		amount_display: (amount / 100).toFixed(2),
		kind: 'manual',
		description: null,
		category_id: 'cat',
		subcategory_id: null,
		transaction_date: '2026-01-01T12:00:00Z',
		created_at: '2026-01-01T12:00:00Z',
		updated_at: '2026-01-01T12:00:00Z'
	};
}

describe('attachParsedReceiptToTransaction', () => {
	beforeEach(() => {
		vi.mocked(attachReceipt).mockReset();
	});

	it('attaches without prompt when sums match', async () => {
		vi.mocked(attachReceipt).mockResolvedValue({
			receipt_import_id: 'r1',
			transaction: tx(5000)
		});
		const prompt = vi.fn();
		const result = await attachParsedReceiptToTransaction(tx(5000), receipt, prompt);
		expect(result.amount).toBe(5000);
		expect(prompt).not.toHaveBeenCalled();
		expect(attachReceipt).toHaveBeenCalledWith(
			expect.objectContaining({ acknowledge_sum_mismatch: false })
		);
	});

	it('prompts and sets acknowledge when sums differ', async () => {
		vi.mocked(attachReceipt).mockResolvedValue({
			receipt_import_id: 'r1',
			transaction: tx(4000)
		});
		const prompt = vi.fn().mockResolvedValue(true);
		await attachParsedReceiptToTransaction(tx(4000), receipt, prompt);
		expect(prompt).toHaveBeenCalledOnce();
		expect(attachReceipt).toHaveBeenCalledWith(
			expect.objectContaining({ acknowledge_sum_mismatch: true })
		);
	});

	it('aborts when user declines mismatch', async () => {
		const prompt = vi.fn().mockResolvedValue(false);
		await expect(attachParsedReceiptToTransaction(tx(1000), receipt, prompt)).rejects.toMatchObject(
			{ code: 'ERR_ABORTED' }
		);
		expect(attachReceipt).not.toHaveBeenCalled();
	});

	it('retries with acknowledge after ERR_RECEIPT_SUM_MISMATCH', async () => {
		vi.mocked(attachReceipt)
			.mockRejectedValueOnce(new ApiError('ERR_RECEIPT_SUM_MISMATCH', 'mismatch', 409))
			.mockResolvedValueOnce({
				receipt_import_id: 'r1',
				transaction: tx(4000)
			});
		const prompt = vi.fn().mockResolvedValue(true);
		await attachParsedReceiptToTransaction(tx(4000), receipt, prompt);
		expect(attachReceipt).toHaveBeenCalledTimes(2);
		expect(attachReceipt).toHaveBeenLastCalledWith(
			expect.objectContaining({ acknowledge_sum_mismatch: true })
		);
	});
});
