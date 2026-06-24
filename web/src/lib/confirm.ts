import { writable } from 'svelte/store';

export type ConfirmOptions = {
	title?: string;
	message: string;
	confirmLabel?: string;
	cancelLabel?: string;
	/** Destructive action — red accent on confirm button */
	danger?: boolean;
};

export type ConfirmState = {
	open: boolean;
	options: ConfirmOptions;
};

const closed: ConfirmState = {
	open: false,
	options: { message: '' }
};

export const confirmStore = writable<ConfirmState>(closed);

let pendingResolve: ((value: boolean) => void) | null = null;

/** In-page confirmation dialog (replaces window.confirm). */
export function confirm(options: ConfirmOptions | string): Promise<boolean> {
	const opts = typeof options === 'string' ? { message: options } : options;
	return new Promise((resolve) => {
		if (pendingResolve) {
			pendingResolve(false);
		}
		pendingResolve = resolve;
		confirmStore.set({ open: true, options: opts });
	});
}

export function resolveConfirm(value: boolean) {
	confirmStore.update((state) => ({ ...state, open: false }));
	pendingResolve?.(value);
	pendingResolve = null;
}
