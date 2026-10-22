import { writable } from 'svelte/store';

export type PromptOptions = {
	title?: string;
	message?: string;
	defaultValue?: string;
	confirmLabel?: string;
	cancelLabel?: string;
	maxLength?: number;
};

export type PromptState = {
	open: boolean;
	options: PromptOptions;
};

const closed: PromptState = {
	open: false,
	options: {}
};

export const promptStore = writable<PromptState>(closed);

let pendingResolve: ((value: string | null) => void) | null = null;

/** In-page text prompt (replaces window.prompt). Resolves null on cancel. */
export function promptText(options: PromptOptions | string): Promise<string | null> {
	const opts = typeof options === 'string' ? { message: options } : options;
	return new Promise((resolve) => {
		if (pendingResolve) {
			pendingResolve(null);
		}
		pendingResolve = resolve;
		promptStore.set({ open: true, options: opts });
	});
}

export function resolvePrompt(value: string | null) {
	promptStore.set(closed);
	if (pendingResolve) {
		pendingResolve(value);
		pendingResolve = null;
	}
}
