/** Reorder list items by moving one id before another. */
export function moveId(ids: readonly string[], fromId: string, toId: string): string[] | null {
	if (fromId === toId) return null;
	const from = ids.indexOf(fromId);
	const to = ids.indexOf(toId);
	if (from < 0 || to < 0) return null;
	const next = [...ids];
	next.splice(from, 1);
	next.splice(to, 0, fromId);
	return next;
}

export type DragGhostView = {
	x: number;
	y: number;
	offsetX: number;
	offsetY: number;
	width: number;
	height: number;
	html: string;
};

type PointerDragOptions = {
	e: PointerEvent;
	id: string;
	rowEl: HTMLElement;
	dragKind: string;
	previewSelector?: string;
	isDisabled?: () => boolean;
	setGhost: (ghost: DragGhostView | null) => void;
	setDraggingId: (id: string | null) => void;
	setOverId: (id: string | null) => void;
	onDrop: (fromId: string, toId: string) => void;
};

function findDragRowId(x: number, y: number, excludeId: string, dragKind: string): string | null {
	for (const el of document.elementsFromPoint(x, y)) {
		const row = el.closest('[data-drag-id]') as HTMLElement | null;
		if (!row?.dataset.dragId || row.dataset.dragId === excludeId) continue;
		if (row.dataset.dragKind !== dragKind) continue;
		return row.dataset.dragId;
	}
	return null;
}

/** Pointer drag with a floating preview that follows the cursor. */
export function beginPointerDrag(opts: PointerDragOptions) {
	if (opts.isDisabled?.()) return;
	if (opts.e.button !== 0) return;
	opts.e.preventDefault();

	const handle = opts.e.currentTarget as HTMLElement;
	handle.setPointerCapture(opts.e.pointerId);

	const cardRect = opts.rowEl.getBoundingClientRect();
	const rowLine = opts.rowEl.querySelector(
		opts.previewSelector ?? '[data-drag-row]'
	) as HTMLElement | null;

	let html: string;
	const width = cardRect.width;
	let height = cardRect.height;

	if (opts.dragKind === 'category' && rowLine) {
		const style = getComputedStyle(opts.rowEl);
		const padY = parseFloat(style.paddingTop) + parseFloat(style.paddingBottom);
		height = rowLine.offsetHeight + padY;
		html = `<div class="card box-border ring-2" style="width:${width}px;min-width:${width}px;max-width:${width}px;margin:0;--tw-ring-color:var(--primary)">${rowLine.outerHTML}</div>`;
	} else {
		html = opts.rowEl.outerHTML;
	}

	const offsetX = opts.e.clientX - cardRect.left;
	const offsetY = opts.e.clientY - cardRect.top;

	const paintGhost = (x: number, y: number) => {
		opts.setGhost({ x, y, offsetX, offsetY, width, height, html });
	};

	opts.setDraggingId(opts.id);
	paintGhost(opts.e.clientX, opts.e.clientY);
	document.body.style.userSelect = 'none';
	document.body.style.cursor = 'grabbing';

	const onMove = (ev: PointerEvent) => {
		paintGhost(ev.clientX, ev.clientY);
		opts.setOverId(findDragRowId(ev.clientX, ev.clientY, opts.id, opts.dragKind));
	};

	const finish = (ev: PointerEvent) => {
		if (handle.hasPointerCapture(ev.pointerId)) {
			handle.releasePointerCapture(ev.pointerId);
		}
		const over = findDragRowId(ev.clientX, ev.clientY, opts.id, opts.dragKind);
		const from = opts.id;
		opts.setDraggingId(null);
		opts.setGhost(null);
		opts.setOverId(null);
		document.body.style.userSelect = '';
		document.body.style.cursor = '';
		handle.removeEventListener('pointermove', onMove);
		handle.removeEventListener('pointerup', finish);
		handle.removeEventListener('pointercancel', finish);
		if (over) opts.onDrop(from, over);
	};

	handle.addEventListener('pointermove', onMove);
	handle.addEventListener('pointerup', finish);
	handle.addEventListener('pointercancel', finish);
}
