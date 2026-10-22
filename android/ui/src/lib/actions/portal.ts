export function portal(
	node: HTMLElement,
	target: string | HTMLElement | null | undefined = 'body'
) {
	if (!target) return;

	const dest = typeof target === 'string' ? document.querySelector(target) : target;
	if (!dest) return;

	dest.appendChild(node);
	return {
		destroy() {
			node.remove();
		}
	};
}
