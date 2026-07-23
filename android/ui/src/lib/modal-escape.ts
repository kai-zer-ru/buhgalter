const stack: Array<() => void> = [];
let attached = false;

function onKeydown(e: KeyboardEvent) {
	if (e.key !== 'Escape' || stack.length === 0) return;
	e.preventDefault();
	e.stopPropagation();
	stack[stack.length - 1]();
}

function attach() {
	if (attached || typeof window === 'undefined') return;
	window.addEventListener('keydown', onKeydown, true);
	attached = true;
}

/** Register a modal layer; only the topmost layer receives Escape. */
export function pushModalEscape(close: () => void): () => void {
	stack.push(close);
	attach();
	return () => {
		const index = stack.lastIndexOf(close);
		if (index >= 0) stack.splice(index, 1);
	};
}

export function hasOpenModals(): boolean {
	return stack.length > 0;
}

/** Invoke the topmost modal close handler (hardware back). Returns true if handled. */
export function popTopModal(): boolean {
	const close = stack.pop();
	if (!close) return false;
	close();
	return true;
}
