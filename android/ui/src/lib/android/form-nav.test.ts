import { beforeEach, describe, expect, it, vi } from 'vitest';

const goto = vi.fn(() => Promise.resolve());

vi.mock('$app/navigation', () => ({
	goto: (...args: unknown[]) => goto(...args)
}));

vi.mock('$app/paths', () => ({
	resolve: (path: string) => path
}));

describe('gotoReplace', () => {
	beforeEach(() => {
		goto.mockClear();
	});

	it('calls goto with replaceState', async () => {
		const { gotoReplace } = await import('./form-nav');
		await gotoReplace('/credits/abc');
		expect(goto).toHaveBeenCalledWith('/credits/abc', { replaceState: true });
	});
});

describe('leaveForm', () => {
	beforeEach(() => {
		goto.mockClear();
		vi.resetModules();
	});

	it('pops history when back leaves the form', async () => {
		const back = vi.fn();
		let path = '/transactions/x/edit';
		vi.stubGlobal('window', {
			location: {
				get pathname() {
					return path.split('?')[0];
				},
				get search() {
					return path.includes('?') ? path.slice(path.indexOf('?')) : '';
				}
			},
			history: {
				back() {
					back();
					path = '/accounts/1';
				}
			},
			requestAnimationFrame(cb: FrameRequestCallback) {
				cb(0);
				return 0;
			}
		});

		const { leaveForm } = await import('./form-nav');
		await leaveForm('/accounts/1');

		expect(back).toHaveBeenCalled();
		expect(goto).not.toHaveBeenCalled();
		vi.unstubAllGlobals();
	});

	it('falls back to gotoReplace when still on the form', async () => {
		const back = vi.fn();
		const path = '/transactions/x/edit';
		vi.stubGlobal('window', {
			location: {
				pathname: path,
				search: ''
			},
			history: { back },
			requestAnimationFrame(cb: FrameRequestCallback) {
				cb(0);
				return 0;
			}
		});

		const { leaveForm } = await import('./form-nav');
		await leaveForm('/accounts/1');

		expect(back).toHaveBeenCalled();
		expect(goto).toHaveBeenCalledWith('/accounts/1', { replaceState: true });
		vi.unstubAllGlobals();
	});
});

describe('credit create-nav', () => {
	beforeEach(() => {
		goto.mockClear();
		vi.resetModules();
	});

	it('navigates steps and abandon with replaceState', async () => {
		vi.doMock('$app/navigation', () => ({
			goto: (...args: unknown[]) => goto(...args)
		}));
		vi.doMock('$app/paths', () => ({
			resolve: (path: string) => path
		}));
		const nav = await import('$lib/credits/create-nav');
		nav.goCreditCreateStep('options', '/credits');
		expect(goto).toHaveBeenCalledWith(expect.stringContaining('/credits/new/options'), {
			replaceState: true
		});
		goto.mockClear();
		nav.abandonCreditCreate('/credits');
		expect(goto).toHaveBeenCalledWith('/credits', { replaceState: true });
	});
});
