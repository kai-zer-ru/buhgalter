type PickerInput = Pick<HTMLInputElement, 'disabled' | 'readOnly'> & {
	showPicker?: () => void;
};

/**
 * Open the system picker for `input`. Android WebView only opens it from the
 * clock indicator; a tap on the digits just focuses a segment.
 * Returns true when `showPicker` was called without throwing.
 */
export function openNativePicker(input: PickerInput): boolean {
	if (input.disabled || input.readOnly) return false;
	if (typeof input.showPicker !== 'function') return false;
	try {
		input.showPicker();
		return true;
	} catch {
		// NotAllowedError / InvalidStateError — the field stays a plain input.
		return false;
	}
}

/** `use:nativeTimePicker` on `<input type="time">`: tap anywhere opens the picker. */
export function nativeTimePicker(node: HTMLInputElement) {
	const onClick = () => {
		openNativePicker(node);
	};
	node.addEventListener('click', onClick);
	return {
		destroy() {
			node.removeEventListener('click', onClick);
		}
	};
}
