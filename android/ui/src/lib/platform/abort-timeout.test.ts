import { describe, expect, it, vi } from 'vitest';
import { abortTimeout } from './abort-timeout';

describe('abortTimeout', () => {
	it('aborts after the given delay when AbortSignal.timeout is unavailable', () => {
		vi.useFakeTimers();
		const original = AbortSignal.timeout;
		// Simulate older WebView
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		(AbortSignal as any).timeout = undefined;
		try {
			const signal = abortTimeout(1_000);
			expect(signal.aborted).toBe(false);
			vi.advanceTimersByTime(1_000);
			expect(signal.aborted).toBe(true);
		} finally {
			AbortSignal.timeout = original;
			vi.useRealTimers();
		}
	});
});
