import { describe, expect, it } from 'vitest';
import { moveId } from '$lib/drag-reorder';
import {
	addRow,
	createLinkPreset,
	emptyInlineKeyboard,
	linkPresetsForTrigger,
	moveButton,
	moveRowByIds,
	normalizeInlineKeyboard,
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

describe('toEditable / toApi', () => {
	it('round-trips and strips client ids', () => {
		const api = {
			rows: [[{ type: 'url', text: 'Open', url: '{debt_url}' }]]
		};
		const editable = toEditableKeyboard(api);
		expect(editable.rows[0][0].id).toBeTruthy();
		const out = toApiKeyboard(editable);
		expect(out).toEqual(api);
		expect(out.rows[0][0]).not.toHaveProperty('id');
		expect(JSON.stringify(out)).not.toMatch(/"id"/);
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
