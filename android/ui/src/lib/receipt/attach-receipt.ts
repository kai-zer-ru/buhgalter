import { ApiError, attachReceipt, type ParsedReceipt, type Transaction } from '$lib/api/client';

export type SumMismatchPrompt = (opts: {
	transactionAmount: number;
	receiptTotalSum: number;
}) => Promise<boolean>;

/** Attach parsed receipt to an expense; prompts when sums differ (client or 409). */
type AttachItems = Array<{
	index: number;
	category_id?: string;
	subcategory_id?: string | null;
	include: boolean;
}>;

export async function attachParsedReceiptToTransaction(
	tx: Transaction,
	receipt: ParsedReceipt,
	promptSumMismatch: SumMismatchPrompt,
	items: AttachItems = []
): Promise<Transaction> {
	let acknowledge = false;
	if (tx.amount !== receipt.total_sum) {
		const ok = await promptSumMismatch({
			transactionAmount: tx.amount,
			receiptTotalSum: receipt.total_sum
		});
		if (!ok) {
			throw new ApiError('ERR_ABORTED', 'Aborted', 0);
		}
		acknowledge = true;
	}

	try {
		const res = await attachReceipt({
			transaction_id: tx.id,
			fiscal_drive_number: receipt.fiscal_drive_number,
			fiscal_document_number: receipt.fiscal_document_number,
			fiscal_sign: receipt.fiscal_sign,
			receipt,
			items,
			acknowledge_sum_mismatch: acknowledge
		});
		return res.transaction;
	} catch (err) {
		if (err instanceof ApiError && err.code === 'ERR_RECEIPT_SUM_MISMATCH') {
			const ok = await promptSumMismatch({
				transactionAmount: tx.amount,
				receiptTotalSum: receipt.total_sum
			});
			if (!ok) {
				throw new ApiError('ERR_ABORTED', 'Aborted', 0);
			}
			const res = await attachReceipt({
				transaction_id: tx.id,
				fiscal_drive_number: receipt.fiscal_drive_number,
				fiscal_document_number: receipt.fiscal_document_number,
				fiscal_sign: receipt.fiscal_sign,
				receipt,
				items,
				acknowledge_sum_mismatch: true
			});
			return res.transaction;
		}
		throw err;
	}
}
