import { writable } from 'svelte/store';

export type ShellHeaderState = {
	title: string;
	onBack: () => void;
};

/** When set, AndroidShell shows back + title instead of menu + app name. */
export const shellHeader = writable<ShellHeaderState | null>(null);
