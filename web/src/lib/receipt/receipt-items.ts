export type ReceiptRowState = {
	index: number;
	include: boolean;
	categoryId: string;
	subcategoryId: string;
};

export function initReceiptRows(count: number): ReceiptRowState[] {
	return Array.from({ length: count }, (_, index) => ({
		index,
		include: true,
		categoryId: '',
		subcategoryId: ''
	}));
}

export function applyCategoryToUnassigned(
	rows: ReceiptRowState[],
	categoryId: string,
	subcategoryId = ''
): ReceiptRowState[] {
	if (!categoryId) return rows;
	return rows.map((row) =>
		row.include && row.categoryId === '' ? { ...row, categoryId, subcategoryId } : row
	);
}

export function countIncludedRows(rows: ReceiptRowState[]): number {
	return rows.filter((row) => row.include).length;
}

export function includedRowsMissingCategory(rows: ReceiptRowState[]): boolean {
	return rows.some((row) => row.include && !row.categoryId);
}

export function formatReceiptItemSum(kopecks: number): string {
	return (kopecks / 100).toFixed(2);
}
