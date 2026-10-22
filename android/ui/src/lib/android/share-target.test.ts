import { describe, expect, it, beforeEach } from 'vitest';
import {
	descriptionFromShare,
	hasShareContent,
	setSharePrefill,
	takeSharePrefill,
	resetSharePrefillForTests
} from './share-target';

describe('share-target', () => {
	beforeEach(() => {
		resetSharePrefillForTests();
	});

	it('builds description from text and subject', () => {
		expect(descriptionFromShare({ subject: 'Чек', text: 'Кофе 250' }, 'shared image')).toBe(
			'Чек\nКофе 250'
		);
	});

	it('falls back to image label when only stream', () => {
		expect(
			descriptionFromShare({ streamUri: 'content://media/1' }, 'Изображение из «Поделиться»')
		).toBe('Изображение из «Поделиться»');
	});

	it('detects share content', () => {
		expect(hasShareContent({})).toBe(false);
		expect(hasShareContent({ text: 'x' })).toBe(true);
		expect(hasShareContent({ streamUri: 'content://x' })).toBe(true);
	});

	it('set/take prefill once', () => {
		setSharePrefill('  hello  ');
		expect(takeSharePrefill()).toEqual({ description: 'hello' });
		expect(takeSharePrefill()).toBeNull();
	});
});
