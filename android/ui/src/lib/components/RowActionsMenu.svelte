<script lang="ts">
	import { portal } from '$lib/actions/portal';
	import type { IconName } from '$lib/components/IconButton.svelte';
	import IconGlyph from '$lib/components/IconGlyph.svelte';
	import { actionMenuStyle } from '$lib/dropdown-position';
	import { randomId } from '$lib/random-id';
	import { _ } from 'svelte-i18n';

	export type RowAction = {
		icon: IconName;
		label: string;
		variant?: 'default' | 'danger';
		disabled?: boolean;
		onclick: () => void;
	};

	let {
		actions,
		align = 'end'
	}: {
		actions: RowAction[];
		align?: 'start' | 'end';
	} = $props();

	let open = $state(false);
	let triggerEl: HTMLButtonElement | undefined = $state();
	let menuEl: HTMLDivElement | undefined = $state();
	let menuStyle = $state('');
	let highlighted = $state(0);

	const menuId = `row-actions-${randomId()}`;
	const visibleActions = $derived(actions.filter((action) => !action.disabled));

	function positionMenu() {
		if (!triggerEl) return;
		const rowHeight = 40;
		const menuHeight = Math.min(320, Math.max(visibleActions.length, 1) * rowHeight + 8);
		menuStyle = actionMenuStyle(triggerEl, menuHeight, align, menuEl?.offsetWidth);
	}

	function close() {
		open = false;
	}

	function toggle() {
		if (visibleActions.length === 0) return;
		open = !open;
		if (open) {
			highlighted = 0;
			requestAnimationFrame(positionMenu);
		}
	}

	function runAction(action: RowAction) {
		if (action.disabled) return;
		close();
		action.onclick();
		triggerEl?.focus();
	}

	function onDocumentPointerDown(event: PointerEvent) {
		const target = event.target as Node;
		if (triggerEl?.contains(target) || menuEl?.contains(target)) return;
		close();
	}

	function onWindowChange() {
		if (open) positionMenu();
	}

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

	$effect(() => {
		if (!open || !triggerEl || !menuEl) return;
		positionMenu();
		requestAnimationFrame(positionMenu);
	});

	function onTriggerKeydown(event: KeyboardEvent) {
		if (!open && (event.key === 'ArrowDown' || event.key === 'Enter' || event.key === ' ')) {
			event.preventDefault();
			open = true;
			requestAnimationFrame(positionMenu);
			return;
		}
		if (!open) return;

		if (event.key === 'Escape') {
			event.preventDefault();
			close();
			return;
		}
		if (event.key === 'ArrowDown') {
			event.preventDefault();
			highlighted = Math.min(highlighted + 1, visibleActions.length - 1);
		}
		if (event.key === 'ArrowUp') {
			event.preventDefault();
			highlighted = Math.max(highlighted - 1, 0);
		}
		if (event.key === 'Enter') {
			event.preventDefault();
			const action = visibleActions[highlighted];
			if (action) runAction(action);
		}
	}
</script>

{#if visibleActions.length > 0}
	<div class="relative inline-flex">
		<button
			type="button"
			bind:this={triggerEl}
			class="btn-icon btn-ghost"
			aria-label={$_('common.actions')}
			aria-expanded={open}
			aria-controls={menuId}
			aria-haspopup="menu"
			onclick={toggle}
			onkeydown={onTriggerKeydown}
		>
			<IconGlyph icon="more-vertical" />
		</button>
		{#if open}
			<div
				bind:this={menuEl}
				id={menuId}
				class="popover-panel min-w-[11rem] overflow-hidden py-1"
				style={menuStyle}
				role="menu"
				use:portal={document.body}
			>
				{#each visibleActions as action, index (action.label + action.icon)}
					<button
						type="button"
						class="flex w-full cursor-pointer items-center gap-2.5 px-3 py-2 text-left text-sm"
						class:opacity-50={action.disabled}
						style:color={action.variant === 'danger' ? 'var(--danger)' : 'var(--text)'}
						style:background-color={index === highlighted
							? 'color-mix(in srgb, var(--primary) 12%, transparent)'
							: 'transparent'}
						disabled={action.disabled}
						role="menuitem"
						onmousedown={(event) => event.preventDefault()}
						onmouseenter={() => (highlighted = index)}
						onclick={() => runAction(action)}
					>
						<IconGlyph icon={action.icon} class="h-4 w-4 shrink-0" />
						<span>{action.label}</span>
					</button>
				{/each}
			</div>
		{/if}
	</div>
{/if}
