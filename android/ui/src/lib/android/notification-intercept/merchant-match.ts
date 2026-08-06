export type MerchantRef = {
	id: string;
	name: string;
};

function norm(s: string): string {
	return s.trim().toLowerCase().replace(/\s+/g, ' ');
}

/**
 * Case-insensitive exact / contains match against merchant catalog.
 * Does not create merchants.
 */
export function matchMerchant(
	merchantText: string,
	merchants: MerchantRef[]
): { merchantId?: string; merchantName?: string } {
	const q = norm(merchantText);
	if (!q) return {};

	const exact = merchants.find((m) => norm(m.name) === q);
	if (exact) return { merchantId: exact.id, merchantName: exact.name };

	const contains = merchants.find((m) => {
		const n = norm(m.name);
		return n.includes(q) || q.includes(n);
	});
	if (contains) return { merchantId: contains.id, merchantName: contains.name };

	return { merchantName: merchantText.trim().slice(0, 120) };
}
