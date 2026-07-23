import { get } from 'svelte/store';
import { page } from '$app/stores';
import { dismissConfirm } from '$lib/confirm';
import { tr } from '$lib/i18n';
import { popTopModal } from '$lib/modal-escape';
import { toast } from '$lib/toast';

export type AndroidBackHandlerContext = {
	isDrawerOpen: () => boolean;
	closeDrawer: () => void;
};

export const EXIT_CONFIRM_MS = 2000;

export type RootBackResult = 'history' | 'prompt' | 'exit';

/** Home dashboard — always double back to exit (ignores WebView history). */
export function resolveRootBackPress(
	path: string,
	canGoBack: boolean,
	pendingExitAt: number,
	now: number,
	windowMs = EXIT_CONFIRM_MS
): { result: RootBackResult; pendingExitAt: number } {
	if (path === '/') {
		if (pendingExitAt > 0 && now - pendingExitAt < windowMs) {
			return { result: 'exit', pendingExitAt: 0 };
		}
		return { result: 'prompt', pendingExitAt: now };
	}
	if (canGoBack) {
		return { result: 'history', pendingExitAt: 0 };
	}
	return { result: 'exit', pendingExitAt: 0 };
}

let removeListener: (() => void) | null = null;
let pendingExitAt = 0;

export async function initAndroidBackHandler(ctx: AndroidBackHandlerContext): Promise<() => void> {
	if (removeListener) {
		removeListener();
		removeListener = null;
	}
	pendingExitAt = 0;

	const { App } = await import('@capacitor/app');
	const sub = await App.addListener('backButton', ({ canGoBack }) => {
		if (ctx.isDrawerOpen()) {
			pendingExitAt = 0;
			ctx.closeDrawer();
			return;
		}
		if (popTopModal()) {
			pendingExitAt = 0;
			return;
		}
		if (dismissConfirm()) {
			pendingExitAt = 0;
			return;
		}

		const path = get(page).url.pathname;
		const { result, pendingExitAt: nextPending } = resolveRootBackPress(
			path,
			canGoBack,
			pendingExitAt,
			Date.now()
		);
		pendingExitAt = nextPending;

		if (result === 'history') {
			window.history.back();
			return;
		}
		if (result === 'prompt') {
			toast(tr('app.exitConfirm'), 'info');
			return;
		}
		void App.exitApp();
	});

	removeListener = () => {
		void sub.remove();
		removeListener = null;
		pendingExitAt = 0;
	};
	return removeListener;
}

/** @internal test helper */
export function resetExitBackStateForTests(): void {
	pendingExitAt = 0;
}
