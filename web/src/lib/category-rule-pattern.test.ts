import { describe, expect, it } from 'vitest';
import { defaultPatternFromDescription } from './category-rule-pattern';

describe('defaultPatternFromDescription', () => {
	it('trims and collapses spaces', () => {
		expect(defaultPatternFromDescription('  foo   bar  ')).toBe('foo bar');
	});
});
