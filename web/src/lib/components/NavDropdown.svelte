<script lang="ts">
	import { afterNavigate } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/stores';
	import { _ } from 'svelte-i18n';
	import { portal } from '$lib/actions/portal';
	import { actionMenuStyle } from '$lib/dropdown-position';
	import type { BackLinkHref } from '$lib/components/BackLink.svelte';

	export type NavDropdownItem = {
		path: BackLinkHref;
		labelKey: string;
		isActive?: (pathname: string) => boolean;
	};

	let {
		labelKey,
		items,
		isGroupActive
	}: {
		labelKey: string;
		items: NavDropdownItem[];
		isGroupActive: (pathname: string) => boolean;
	} = $props();

	let open = $state(false);
	let triggerEl: HTMLButtonElement | undefined = $state();
	let menuEl: HTMLDivElement | undefined = $state();
	let menuStyle = $state('');

	const pathname = $derived($page.url.pathname);
	const groupActive = $derived(isGroupActive(pathname));

	function positionMenu() {
		if (!triggerEl) return;
		const rowHeight = 40;
		const menuHeight = Math.min(320, Math.max(items.length, 1) * rowHeight + 8);
		menuStyle = actionMenuStyle(triggerEl, menuHeight, 'end', menuEl?.offsetWidth);
	}

	function close() {
		open = false;
	}

	function toggle(event: MouseEvent) {
		event.stopPropagation();
		open = !open;
	}

	function itemActive(item: NavDropdownItem) {
		return item.isActive ? item.isActive(pathname) : pathname === item.path;
	}

	function onDocumentPointerDown(event: PointerEvent) {
		const target = event.target as Node;
		if (triggerEl?.contains(target) || menuEl?.contains(target)) return;
		close();
	}

	function onTriggerPointerDown(event: PointerEvent) {
		// Не закрывать меню тем же кликом, что открыл (capture на document срабатывает раньше click)
		event.stopPropagation();
	}

	function onWindowChange() {
		if (open) positionMenu();
	}

	$effect(() => {
		if (!open || !triggerEl || !menuEl) return;
		positionMenu();
		requestAnimationFrame(positionMenu);
	});

	$effect(() => {
		if (!open) return;
		document.addEventListener('pointerdown', onDocumentPointerDown, true);
		window.addEventListener('resize', onWindowChange);
		window.addEventListener('scroll', onWindowChange, true);
		return () => {
			document.removeEventListener('pointerdown', onDocumentPointerDown, true);
			window.removeEventListener('resize', onWindowChange);
			window.removeEventListener('scroll', onWindowChange, true);
		};
	});

	afterNavigate(() => {
		close();
	});

	function handleItemClick() {
		close();
	}

	function onTriggerKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			close();
			return;
		}
		if (event.key === 'ArrowDown' && !open) {
			event.preventDefault();
			open = true;
		}
	}
</script>

<div class="relative">
	<button
		type="button"
		bind:this={triggerEl}
		class="btn-ghost btn-nav inline-flex items-center gap-1 {groupActive ? 'nav-link-active' : ''}"
		aria-expanded={open}
		aria-haspopup="true"
		onclick={toggle}
		onpointerdown={onTriggerPointerDown}
		onkeydown={onTriggerKeydown}
	>
		<span>{$_(labelKey)}</span>
		<svg
			aria-hidden="true"
			class="h-3.5 w-3.5 shrink-0 opacity-70"
			viewBox="0 0 24 24"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
		>
			<path d="M6 9l6 6 6-6" />
		</svg>
	</button>
	{#if open}
		<div
			bind:this={menuEl}
			class="popover-panel min-w-[12rem] p-1"
			style={menuStyle}
			role="menu"
			use:portal={document.body}
		>
			{#each items as item (item.path)}
				<a
					href={resolve(item.path)}
					class="block rounded-lg px-3 py-2 text-sm hover:opacity-90 {itemActive(item)
						? 'nav-link-active'
						: ''}"
					role="menuitem"
					aria-current={itemActive(item) ? 'page' : undefined}
					onclick={handleItemClick}
				>
					{$_(item.labelKey)}
				</a>
			{/each}
		</div>
	{/if}
</div>
