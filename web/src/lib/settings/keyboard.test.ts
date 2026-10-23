import { describe, expect, it } from 'vitest';
import { moveId } from '$lib/drag-reorder';
import {
	addRow,
	composeTemplateRows,
	createLinkPreset,
	emptyInlineKeyboard,
	ensureKeyboardIds,
	linkPresetsForTrigger,
	moveButton,
	moveRowByIds,
	normalizeInlineKeyboard,
	splitTemplateKeyboard,
	toApiKeyboard,
	toEditableKeyboard,
	validateInlineKeyboard,
	type EditableKeyboardButton
} from './keyboard';

function btn(
	partial: Partial<EditableKeyboardButton> & Pick<EditableKeyboardButton, 'id' | 'type' | 'text'>
): EditableKeyboardButton {
	return partial as EditableKeyboardButton;
}

describe('normalizeInlineKeyboard', () => {
	it('returns empty rows for nullish / invalid', () => {
		expect(normalizeInlineKeyboard(undefined)).toEqual({ rows: [] });
		expect(normalizeInlineKeyboard(null as never)).toEqual({ rows: [] });
		expect(emptyInlineKeyboard()).toEqual({ rows: [] });
	});

	it('clones rows', () => {
		const src = { rows: [[{ type: 'url', text: 'A', url: '{debt_url}' }]] };
		const next = normalizeInlineKeyboard(src);
		expect(next).toEqual(src);
		expect(next.rows[0][0]).not.toBe(src.rows[0][0]);
	});
});

describe('ensureKeyboardIds', () => {
	it('returns empty rows for nullish', () => {
		expect(ensureKeyboardIds(undefined)).toEqual({ rows: [] });
		expect(ensureKeyboardIds(null)).toEqual({ rows: [] });
	});

	it('fills a missing id and keeps an existing one', () => {
		const next = ensureKeyboardIds({
			rows: [
				[
					{ type: 'url', text: 'A', url: 'https://example.com' },
					{ id: 'keep', type: 'url', text: 'B', url: 'https://example.com' }
				]
			]
		});
		expect(next.rows[0][0].id).toMatch(/^btn-/);
		expect(next.rows[0][1].id).toBe('keep');
	});
});

describe('toEditable / toApi', () => {
	it('keeps stored ids', () => {
		const api = {
			rows: [[{ id: 'g1', type: 'url', text: 'Open', url: '{debt_url}' }]]
		};
		const editable = toEditableKeyboard(api);
		expect(editable.rows[0][0].id).toBe('g1');
		const out = toApiKeyboard(editable);
		expect(out.rows[0][0].id).toBe('g1');
		expect(out.rows[0][0]).toMatchObject({ type: 'url', text: 'Open', url: '{debt_url}' });
	});
});

describe('composeTemplateRows', () => {
	it('shows globals before locals when placement is absent', () => {
		const rows = composeTemplateRows(
			{ rows: [[{ id: 'g1', type: 'url', text: 'G', url: 'https://g' }]] },
			{ rows: [[{ id: 'l1', type: 'url', text: 'L', url: 'https://l' }]] }
		);
		expect(rows.map((row) => row.map((btn) => [btn.id, btn.locked]))).toEqual([
			[['g1', true]],
			[['l1', undefined]]
		]);
	});

	it('follows placement and appends a new global button', () => {
		const rows = composeTemplateRows(
			{
				rows: [
					[
						{ id: 'g1', type: 'url', text: 'G1', url: 'https://a' },
						{ id: 'g2', type: 'url', text: 'G2', url: 'https://b' }
					]
				]
			},
			{
				rows: [[{ id: 'l1', type: 'url', text: 'L', url: 'https://l' }]],
				placement: [['l:l1', 'g:g1']]
			}
		);
		expect(rows.map((row) => row.map((btn) => btn.id))).toEqual([['l1', 'g1'], ['g2']]);
		expect(rows[0][1].locked).toBe(true);
		expect(rows[1][0].text).toBe('G2');
	});

	it('split keeps globals out of local rows and round-trips order', () => {
		const visual = composeTemplateRows(
			{ rows: [[{ id: 'g1', type: 'url', text: 'G', url: 'https://g' }]] },
			{
				rows: [[{ id: 'l1', type: 'callback_data', text: 'L', callback_data: 'x' }]],
				placement: [['g:g1', 'l:l1']]
			}
		);
		const stored = splitTemplateKeyboard(visual);
		expect(stored.rows.map((row) => row[0].id)).toEqual(['l1']);
		expect(stored.placement).toEqual([['g:g1', 'l:l1']]);
		expect(JSON.stringify(stored)).not.toMatch(/"locked"/);
		const again = composeTemplateRows(
			{ rows: [[{ id: 'g1', type: 'url', text: 'G', url: 'https://g' }]] },
			stored
		);
		expect(again.map((row) => row.map((btn) => btn.id))).toEqual([['g1', 'l1']]);
	});
});

describe('moveButton / moveRowByIds', () => {
	it('reorders buttons across rows', () => {
		const a = btn({ id: 'a', type: 'url', text: 'A', url: 'https://a' });
		const b = btn({ id: 'b', type: 'url', text: 'B', url: 'https://b' });
		const c = btn({ id: 'c', type: 'url', text: 'C', url: 'https://c' });
		const next = moveButton([[a, b], [c]], 'a', 'c');
		expect(next?.map((r) => r.map((x) => x.id))).toEqual([['b'], ['a', 'c']]);
	});

	it('reorders rows via moveId helper', () => {
		const ids = ['r0', 'r1', 'r2'];
		expect(moveId(ids, 'r0', 'r2')).toEqual(['r1', 'r2', 'r0']);
		const rows = [
			[btn({ id: 'a', type: 'url', text: 'A', url: 'x' })],
			[btn({ id: 'b', type: 'url', text: 'B', url: 'x' })],
			[btn({ id: 'c', type: 'url', text: 'C', url: 'x' })]
		];
		const next = moveRowByIds(rows, ['r0', 'r1', 'r2'], 'r0', 'r2');
		expect(next?.map((r) => r[0].id)).toEqual(['b', 'c', 'a']);
	});
});

describe('validateInlineKeyboard', () => {
	it('accepts empty keyboard', () => {
		expect(validateInlineKeyboard('telegram', emptyInlineKeyboard())).toBeNull();
		expect(validateInlineKeyboard('max', emptyInlineKeyboard())).toBeNull();
	});

	it('requires text and url for telegram url button', () => {
		expect(
			validateInlineKeyboard('telegram', {
				rows: [[{ type: 'url', text: '', url: '{debt_url}' }]]
			})?.code
		).toBe('text_required');
		expect(
			validateInlineKeyboard('telegram', {
				rows: [[{ type: 'url', text: 'Open', url: '' }]]
			})?.code
		).toBe('url_required');
	});

	it('enforces telegram per-row limit', () => {
		const row = Array.from({ length: 9 }, (_, i) => ({
			type: 'url',
			text: `B${i}`,
			url: 'https://example.com'
		}));
		expect(validateInlineKeyboard('telegram', { rows: [row] })?.code).toBe('max_buttons_per_row');
	});

	it('enforces max special row limit', () => {
		const row = [
			{ type: 'link', text: 'A', url: 'https://a' },
			{ type: 'link', text: 'B', url: 'https://b' },
			{ type: 'link', text: 'C', url: 'https://c' },
			{ type: 'callback', text: 'D', payload: 'x' }
		];
		expect(validateInlineKeyboard('max', { rows: [row] })?.code).toBe('max_special_per_row');
	});
});

describe('linkPresetsForTrigger / createLinkPreset / addRow', () => {
	it('maps credit and shortfall to matching url presets', () => {
		expect(linkPresetsForTrigger('credit_payment')).toEqual([
			expect.objectContaining({ url: '{credit_url}' })
		]);
		expect(linkPresetsForTrigger('balance_shortfall')).toEqual([
			expect.objectContaining({ url: '{account_url}' })
		]);
		expect(linkPresetsForTrigger('test')).toEqual([
			expect.objectContaining({ url: '{settings_url}' })
		]);
	});

	it('returns several presets for global (null)', () => {
		const urls = linkPresetsForTrigger(null).map((p) => p.url);
		expect(urls).toEqual([
			'{debt_url}',
			'{credit_url}',
			'{transaction_url}',
			'{budget_url}',
			'{settings_url}'
		]);
	});

	it('createLinkPreset builds url/link button for provider', () => {
		const def = linkPresetsForTrigger('credit_payment')[0];
		expect(createLinkPreset('telegram', def)).toMatchObject({
			type: 'url',
			url: '{credit_url}'
		});
		expect(createLinkPreset('max', def)).toMatchObject({
			type: 'link',
			url: '{credit_url}'
		});
	});

	it('addRow appends a default button row', () => {
		const next = addRow([], 'telegram');
		expect(next).toHaveLength(1);
		expect(next[0][0].type).toBe('url');
	});
});
