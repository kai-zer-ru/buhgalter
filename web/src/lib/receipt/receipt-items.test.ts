import { describe, expect, it } from 'vitest';
import {
	applyCategoryToSelected,
	countIncludedRows,
	includedRowsMissingCategory,
	initReceiptRows
} from './receipt-items';

describe('receipt-items', () => {
	it('initReceiptRows creates included rows', () => {
		const rows = initReceiptRows(3);
		expect(rows).toHaveLength(3);
		expect(rows.every((row) => row.include && !row.selected)).toBe(true);
	});

	it('applyCategoryToSelected updates only selected rows', () => {
		const rows = initReceiptRows(3);
		rows[0].selected = true;
		rows[2].selected = true;
		const updated = applyCategoryToSelected(rows, 'cat-1', 'sub-1');
		expect(updated[0].categoryId).toBe('cat-1');
		expect(updated[0].subcategoryId).toBe('sub-1');
		expect(updated[0].selected).toBe(false);
		expect(updated[1].categoryId).toBe('');
		expect(updated[2].categoryId).toBe('cat-1');
	});

	it('countIncludedRows and missing category', () => {
		const rows = initReceiptRows(2);
		rows[0].categoryId = 'cat';
		rows[1].include = false;
		expect(countIncludedRows(rows)).toBe(1);
		expect(includedRowsMissingCategory(rows)).toBe(false);
		rows[1].include = true;
		expect(includedRowsMissingCategory(rows)).toBe(true);
	});
});
