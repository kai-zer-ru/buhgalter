import { readFileSync } from 'node:fs';
import { fileURLToPath } from 'node:url';
import { dirname, join } from 'node:path';
import { describe, expect, it } from 'vitest';

const componentPath = join(dirname(fileURLToPath(import.meta.url)), 'TransactionList.svelte');

/** Desktop table + mobile cards: one MoneyDisplay per row layout. */
describe('TransactionList amount cell', () => {
	it('renders tx.amount_display once per desktop and mobile row', () => {
		const src = readFileSync(componentPath, 'utf8');
		const matches = src.match(/value=\{tx\.amount_display\}/g);
		expect(matches?.length).toBe(2);
	});
});
