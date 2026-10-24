export type MerchantRef = {
	id: string;
	name: string;
	aliases?: string[];
};

function norm(s: string): string {
	return s.trim().toLowerCase().replace(/\s+/g, ' ');
}

function namesOf(merchant: MerchantRef): string[] {
	return [merchant.name, ...(merchant.aliases ?? [])];
}

/** Catalog rows for intercept matching. Keeps aliases; drops blank ones. */
export function toMerchantRefs(
	rows: { id: string; name: string; aliases?: string[] }[]
): MerchantRef[] {
	return rows.map((row) => ({
		id: row.id,
		name: row.name,
		aliases: (row.aliases ?? []).map((alias) => alias.trim()).filter((alias) => alias !== '')
	}));
}

/**
 * Case-insensitive exact / contains match against the primary name and aliases.
 * Does not create merchants. The returned name is always the primary name.
 */
export function matchMerchant(
	merchantText: string,
	merchants: MerchantRef[]
): { merchantId?: string; merchantName?: string } {
	const q = norm(merchantText);
	if (!q) return {};

	const exact = merchants.find((m) => namesOf(m).some((name) => norm(name) === q));
	if (exact) return { merchantId: exact.id, merchantName: exact.name };

	const contains = merchants.find((m) =>
		namesOf(m).some((name) => {
			const n = norm(name);
			return n.includes(q) || q.includes(n);
		})
	);
	if (contains) return { merchantId: contains.id, merchantName: contains.name };

	return { merchantName: merchantText.trim().slice(0, 120) };
}
