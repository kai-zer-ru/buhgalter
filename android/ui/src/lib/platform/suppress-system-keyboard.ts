import { App } from '@capacitor/app';
import { hideSoftKeyboard } from '$lib/platform/app-instance';
import { isNativeApp } from '$lib/platform/native';

type VirtualKeyboardNavigator = Navigator & {
	virtualKeyboard?: { hide?: () => void };
};

/**
 * Keep the system IME away from MoneyInput while the in-app keypad is open.
 * `inputmode="none"` alone is not enough on some OEM WebViews after app switch.
 */
export function suppressSystemKeyboard(input?: HTMLInputElement | null): void {
	if (input) {
		input.setAttribute('inputmode', 'none');
		input.setAttribute('virtualkeyboardpolicy', 'manual');
	}
	try {
		(navigator as VirtualKeyboardNavigator).virtualKeyboard?.hide?.();
	} catch {
		// VirtualKeyboard API not available
	}
	if (isNativeApp()) {
		void hideSoftKeyboard();
	}
}

/** While `active`, hide IME on resume / when the system tries to show it. */
export function watchSuppressSystemKeyboard(
	active: () => boolean,
	input: () => HTMLInputElement | null | undefined
): () => void {
	if (!isNativeApp()) return () => {};

	const onResume = () => {
		if (!active()) return;
		suppressSystemKeyboard(input());
		queueMicrotask(() => {
			if (active()) suppressSystemKeyboard(input());
		});
		window.setTimeout(() => {
			if (active()) suppressSystemKeyboard(input());
		}, 50);
		window.setTimeout(() => {
			if (active()) suppressSystemKeyboard(input());
		}, 250);
	};

	let removeApp: (() => void) | undefined;
	void App.addListener('appStateChange', ({ isActive }) => {
		if (isActive) onResume();
	}).then((handle) => {
		removeApp = () => {
			void handle.remove();
		};
	});

	const onVisibility = () => {
		if (document.visibilityState === 'visible') onResume();
	};
	document.addEventListener('visibilitychange', onVisibility);

	return () => {
		removeApp?.();
		document.removeEventListener('visibilitychange', onVisibility);
	};
}
