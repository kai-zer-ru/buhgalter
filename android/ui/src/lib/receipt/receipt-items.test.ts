import { describe, expect, it } from 'vitest';
import {
	applyCategoryToUnassigned,
	countIncludedRows,
	includedRowsMissingCategory,
	initReceiptRows
} from './receipt-items';

describe('receipt-items', () => {
	it('initReceiptRows creates included rows', () => {
		const rows = initReceiptRows(3);
		expect(rows).toHaveLength(3);
		expect(rows.every((row) => row.include && row.categoryId === '')).toBe(true);
	});

	it('applyCategoryToUnassigned updates only included rows without category', () => {
		const rows = initReceiptRows(3);
		rows[0].categoryId = 'existing-cat';
		rows[1].include = false;
		const updated = applyCategoryToUnassigned(rows, 'cat-1', 'sub-1');
		expect(updated[0].categoryId).toBe('existing-cat');
		expect(updated[0].subcategoryId).toBe('');
		expect(updated[1].categoryId).toBe('');
		expect(updated[2].categoryId).toBe('cat-1');
		expect(updated[2].subcategoryId).toBe('sub-1');
	});

	it('applyCategoryToUnassigned no-op without categoryId', () => {
		const rows = initReceiptRows(2);
		const updated = applyCategoryToUnassigned(rows, '', 'sub-1');
		expect(updated).toEqual(rows);
	});

	it('applyCategoryToUnassigned clears nothing when all have categories', () => {
		const rows = initReceiptRows(2);
		rows[0].categoryId = 'a';
		rows[1].categoryId = 'b';
		const updated = applyCategoryToUnassigned(rows, 'bulk', 'sub');
		expect(updated[0].categoryId).toBe('a');
		expect(updated[1].categoryId).toBe('b');
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
