import { describe, expect, it } from 'vitest';
import { EXIT_CONFIRM_MS, resolveRootBackPress } from '$lib/android/back-handler';

describe('resolveRootBackPress', () => {
	it('always prompts on home, even when WebView has history', () => {
		expect(resolveRootBackPress('/', true, 0, 10_000)).toEqual({
			result: 'prompt',
			pendingExitAt: 10_000
		});
	});

	it('uses history on other pages when WebView can go back', () => {
		expect(resolveRootBackPress('/accounts', true, 1000, 1500)).toEqual({
			result: 'history',
			pendingExitAt: 0
		});
	});

	it('prompts on first back at home without history', () => {
		const now = 10_000;
		expect(resolveRootBackPress('/', false, 0, now)).toEqual({
			result: 'prompt',
			pendingExitAt: now
		});
	});

	it('exits on second back within window', () => {
		const first = 10_000;
		const second = first + 500;
		const afterFirst = resolveRootBackPress('/', false, 0, first);
		expect(resolveRootBackPress('/', false, afterFirst.pendingExitAt, second)).toEqual({
			result: 'exit',
			pendingExitAt: 0
		});
	});

	it('prompts again after window expires', () => {
		const first = 10_000;
		const late = first + EXIT_CONFIRM_MS + 1;
		const afterFirst = resolveRootBackPress('/', false, 0, first);
		expect(resolveRootBackPress('/', false, afterFirst.pendingExitAt, late)).toEqual({
			result: 'prompt',
			pendingExitAt: late
		});
	});

	it('exits immediately on non-home root without history', () => {
		expect(resolveRootBackPress('/accounts', false, 0, 1000)).toEqual({
			result: 'exit',
			pendingExitAt: 0
		});
	});
});
