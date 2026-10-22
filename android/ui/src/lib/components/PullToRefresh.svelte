<script lang="ts">
	import type { Snippet } from 'svelte';
	import { page } from '$app/stores';
	import {
		attachNonPassiveTouchMove,
		isPullToRefreshActive,
		isPullToRefreshRoute,
		isPtrBlockedByModal,
		revalidateCurrentPath
	} from '$lib/pull-to-refresh';
	import { isNativeApp } from '$lib/platform/native';

	let {
		onRefresh,
		disabled = false,
		children
	}: {
		/** Silent reload of the current screen after soft invalidate. */
		onRefresh: () => void | Promise<void>;
		/** Force-off (e.g. while a page-local modal is open). */
		disabled?: boolean;
		children: Snippet;
	} = $props();

	const PULL_THRESHOLD_PX = 64;
	const PULL_MAX_PX = 96;

	let rootEl = $state<HTMLDivElement | null>(null);
	let pulling = $state(false);
	let refreshing = $state(false);
	let pullDistance = $state(0);
	let armed = $state(false);

	let startY = 0;
	let startX = 0;
	let tracking = false;
	let horizontalLock = false;

	function readEnv(): { isCapacitor: boolean; coarsePointer: boolean; viewportWidth: number } {
		const isCapacitor =
			isNativeApp() ||
			(typeof document !== 'undefined' &&
				document.documentElement.classList.contains('native-app'));
		const coarsePointer =
			typeof window !== 'undefined' &&
			typeof window.matchMedia === 'function' &&
			window.matchMedia('(pointer: coarse)').matches;
		const viewportWidth = typeof window !== 'undefined' ? window.innerWidth : 1024;
		return { isCapacitor, coarsePointer, viewportWidth };
	}

	function ptrAllowed(): boolean {
		if (disabled || refreshing) return false;
		if (!isPullToRefreshRoute($page.url.pathname)) return false;
		if (!isPullToRefreshActive(readEnv())) return false;
		if (isPtrBlockedByModal()) return false;
		return true;
	}

	function findScrollParent(el: HTMLElement | null): HTMLElement | Window {
		let node: HTMLElement | null = el;
		while (node) {
			const style = getComputedStyle(node);
			const oy = style.overflowY;
			if (
				(oy === 'auto' || oy === 'scroll' || oy === 'overlay') &&
				node.scrollHeight > node.clientHeight + 1
			) {
				return node;
			}
			node = node.parentElement;
		}
		return window;
	}

	function scrollTopOf(target: HTMLElement | Window): number {
		if (target === window) {
			return window.scrollY || document.documentElement.scrollTop || 0;
		}
		return (target as HTMLElement).scrollTop;
	}

	function atPageScrollTop(): boolean {
		return scrollTopOf(findScrollParent(rootEl)) <= 0;
	}

	/** Nested overflow region mid-scroll must not arm PTR (surfaces.md). */
	function isNestedScrollerMidScroll(target: EventTarget | null): boolean {
		const pageScroll = findScrollParent(rootEl);
		let node: HTMLElement | null = target instanceof HTMLElement ? target : null;
		while (node && node !== rootEl) {
			const style = getComputedStyle(node);
			const oy = style.overflowY;
			if (
				(oy === 'auto' || oy === 'scroll' || oy === 'overlay') &&
				node.scrollHeight > node.clientHeight + 1 &&
				node !== pageScroll &&
				node.scrollTop > 0
			) {
				return true;
			}
			node = node.parentElement;
		}
		return false;
	}

	function canStartPull(target: EventTarget | null): boolean {
		return atPageScrollTop() && !isNestedScrollerMidScroll(target);
	}

	function onTouchStart(event: TouchEvent) {
		if (!ptrAllowed() || event.touches.length !== 1) return;
		const t = event.touches[0];
		if (!t) return;
		startY = t.clientY;
		startX = t.clientX;
		tracking = canStartPull(event.target);
		horizontalLock = false;
		pulling = false;
		pullDistance = 0;
		armed = false;
	}

	function onTouchMove(event: TouchEvent) {
		if (!tracking || refreshing || event.touches.length !== 1) return;
		const t = event.touches[0];
		if (!t) return;
		const dy = t.clientY - startY;
		const dx = t.clientX - startX;

		if (!pulling) {
			if (Math.abs(dx) > 10 && Math.abs(dx) > Math.abs(dy)) {
				horizontalLock = true;
				tracking = false;
				return;
			}
			if (dy <= 0 || !canStartPull(event.target)) {
				tracking = false;
				return;
			}
			if (!ptrAllowed()) {
				tracking = false;
				return;
			}
			pulling = true;
		}

		if (horizontalLock) return;

		const dampened = Math.min(PULL_MAX_PX, dy * 0.45);
		pullDistance = dampened;
		armed = dampened >= PULL_THRESHOLD_PX;
		if (dy > 8 && event.cancelable) {
			event.preventDefault();
		}
	}

	async function finishPull() {
		const shouldRefresh = armed && ptrAllowed();
		tracking = false;
		pulling = false;
		armed = false;
		pullDistance = 0;
		if (!shouldRefresh) return;

		refreshing = true;
		try {
			revalidateCurrentPath($page.url.pathname);
			await onRefresh();
		} finally {
			refreshing = false;
		}
	}

	function onTouchEnd() {
		if (!tracking && !pulling) return;
		void finishPull();
	}

	function onTouchCancel() {
		tracking = false;
		pulling = false;
		armed = false;
		pullDistance = 0;
	}

	/** Svelte 5 ontouchmove is passive — need preventDefault while pulling. */
	function nonPassiveTouchMove(node: HTMLElement) {
		return { destroy: attachNonPassiveTouchMove(node, onTouchMove) };
	}

	const indicatorOffset = $derived(
		refreshing ? PULL_THRESHOLD_PX * 0.55 : pulling ? pullDistance : 0
	);
	const indicatorVisible = $derived(refreshing || pullDistance > 8);
</script>

<div
	bind:this={rootEl}
	class="pull-to-refresh"
	class:pull-to-refresh-active={indicatorVisible}
	aria-busy={refreshing}
	use:nonPassiveTouchMove
	ontouchstart={onTouchStart}
	ontouchend={onTouchEnd}
	ontouchcancel={onTouchCancel}
>
	<div
		class="pull-to-refresh-indicator"
		class:visible={indicatorVisible}
		class:refreshing
		style:transform="translateY({indicatorOffset}px)"
		aria-hidden={!indicatorVisible}
	>
		<span class="pull-to-refresh-spinner"></span>
	</div>
	<div class="pull-to-refresh-content" style:transform="translateY({indicatorOffset}px)">
		{@render children()}
	</div>
</div>

<style>
	.pull-to-refresh {
		position: relative;
		overscroll-behavior-y: contain;
	}

	.pull-to-refresh-indicator {
		pointer-events: none;
		position: absolute;
		left: 0;
		right: 0;
		top: -2.5rem;
		display: flex;
		justify-content: center;
		align-items: center;
		height: 2.5rem;
		opacity: 0;
		transition: opacity 120ms ease;
		z-index: 5;
	}

	.pull-to-refresh-indicator.visible {
		opacity: 1;
	}

	.pull-to-refresh-spinner {
		width: 1.25rem;
		height: 1.25rem;
		border-radius: 9999px;
		border: 2px solid color-mix(in srgb, var(--text-muted) 35%, transparent);
		border-top-color: var(--accent, var(--text));
	}

	.pull-to-refresh-indicator.refreshing .pull-to-refresh-spinner {
		animation: pull-to-refresh-spin 0.7s linear infinite;
	}

	.pull-to-refresh-content {
		will-change: transform;
	}

	@keyframes pull-to-refresh-spin {
		to {
			transform: rotate(360deg);
		}
	}

	@media (pointer: fine) and (min-width: 769px) {
		/* Desktop fine-pointer: no visual chrome unless Capacitor class (android preview). */
		:global(html:not(.native-app)) .pull-to-refresh-indicator {
			display: none;
		}
	}
</style>
