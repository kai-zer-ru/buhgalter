// See https://svelte.dev/docs/kit/types#app.d.ts
// for information about these interfaces
declare global {
	namespace App {
		// interface Error {}
		// interface Locals {}
		// interface PageData {}
		// interface PageState {}
		// interface Platform {}
	}
	const __APP_VERSION__: string;
}

/** Virtual Keyboard API — not yet in all DOM/Svelte typings. */
declare module 'svelte/elements' {
	// T must match svelte/elements HTMLAttributes<T> for declaration merging.
	// eslint-disable-next-line @typescript-eslint/no-unused-vars -- merge arity
	export interface HTMLAttributes<T> {
		virtualkeyboardpolicy?: 'auto' | 'manual' | undefined | null;
	}
}

export {};
