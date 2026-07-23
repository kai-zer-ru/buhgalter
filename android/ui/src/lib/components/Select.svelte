<script lang="ts">
	import { portal } from '$lib/actions/portal';
	import { dropdownListStyle } from '$lib/dropdown-position';
	import SelectOptionIcon from '$lib/components/SelectOptionIcon.svelte';
	import type { SelectOption } from '$lib/select-options';

	let {
		value = $bindable(''),
		options,
		id = 'select',
		label = '',
		hint = '',
		placeholder = '',
		disabled = false,
		usePortal = false,
		emptyLabel = '',
		controlled = false,
		onchange
	}: {
		value?: string;
		options: SelectOption[];
		id?: string;
		label?: string;
		hint?: string;
		placeholder?: string;
		disabled?: boolean;
		usePortal?: boolean;
		emptyLabel?: string;
		controlled?: boolean;
		onchange?: (value: string) => void;
	} = $props();

	let open = $state(false);
	let triggerEl: HTMLButtonElement | undefined = $state();
	let listEl: HTMLUListElement | undefined = $state();
	let highlighted = $state(0);
	let listStyle = $state('');

	const listId = $derived(`${id}-list`);

	const selectedOption = $derived(options.find((option) => option.value === value));
	const selectedLabel = $derived(selectedOption?.label ?? placeholder);

	const visibleOptions = $derived(
		options.filter((option) => !option.disabled || option.value === value)
	);

	function positionList() {
		if (!triggerEl) return;
		const listHeight = Math.min(224, Math.max(visibleOptions.length, 1) * 40);
		listStyle = dropdownListStyle(triggerEl, listHeight, usePortal);
	}

	function close() {
		open = false;
	}

	function toggle() {
		if (disabled) return;
		open = !open;
		if (open) {
			const index = visibleOptions.findIndex((option) => option.value === value);
			highlighted = index >= 0 ? index : 0;
			requestAnimationFrame(positionList);
		}
	}

	function selectOption(next: string) {
		if (!controlled) {
			value = next;
		}
		onchange?.(next);
		close();
		triggerEl?.focus();
	}

	function onDocumentPointerDown(event: PointerEvent) {
		const target = event.target as Node;
		if (triggerEl?.contains(target) || listEl?.contains(target)) return;
		close();
	}

	function onWindowChange() {
		if (open) positionList();
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

	function onTriggerKeydown(event: KeyboardEvent) {
		if (!open && (event.key === 'ArrowDown' || event.key === 'ArrowUp' || event.key === 'Enter')) {
			event.preventDefault();
			open = true;
			requestAnimationFrame(positionList);
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
			highlighted = Math.min(highlighted + 1, visibleOptions.length - 1);
		}
		if (event.key === 'ArrowUp') {
			event.preventDefault();
			highlighted = Math.max(highlighted - 1, 0);
		}
		if (event.key === 'Enter') {
			event.preventDefault();
			const option = visibleOptions[highlighted];
			if (option && !option.disabled) selectOption(option.value);
		}
	}
</script>

<div>
	{#if label}
		<label class="field-label" for={id}>{label}</label>
	{/if}
	<!-- relative only around the control: non-portal list uses top/bottom:100% -->
	<div class="relative">
		<button
			{id}
			type="button"
			bind:this={triggerEl}
			class="input flex w-full min-w-0 cursor-pointer items-center justify-between gap-2 text-left"
			class:opacity-60={disabled}
			{disabled}
			role="combobox"
			aria-expanded={open}
			aria-controls={listId}
			onclick={toggle}
			onkeydown={onTriggerKeydown}
		>
			<span class="flex min-w-0 items-center gap-2">
				{#if selectedOption?.icon}
					<SelectOptionIcon icon={selectedOption.icon} />
				{/if}
				<span class="min-w-0 truncate" style:color={value ? 'var(--text)' : 'var(--text-muted)'}
					>{selectedLabel}</span
				>
			</span>
			<span class="shrink-0" aria-hidden="true" style:color="var(--text-muted)">▾</span>
		</button>
		{#if open}
			<ul
				bind:this={listEl}
				id={listId}
				class="popover-panel max-h-56 overflow-y-auto {usePortal ? '' : 'absolute z-20 w-full'}"
				style={listStyle}
				role="listbox"
				use:portal={usePortal ? document.body : null}
			>
				{#if visibleOptions.length === 0}
					<li class="px-4 py-2 text-sm" style:color="var(--text-muted)">{emptyLabel}</li>
				{:else}
					{#each visibleOptions as option, index (option.value)}
						<li>
							<button
								type="button"
								class="flex w-full cursor-pointer items-center gap-2 px-4 py-2 text-left text-sm hover:opacity-90"
								class:font-medium={option.value === value}
								style:background-color={index === highlighted || option.value === value
									? 'color-mix(in srgb, var(--primary) 12%, transparent)'
									: 'transparent'}
								disabled={option.disabled}
								onmousedown={(event) => event.preventDefault()}
								onclick={() => selectOption(option.value)}
							>
								{#if option.icon}
									<SelectOptionIcon icon={option.icon} />
								{/if}
								<span class="min-w-0 truncate">{option.label}</span>
							</button>
						</li>
					{/each}
				{/if}
			</ul>
		{/if}
	</div>
	{#if hint}
		<p class="mt-1 text-xs" style:color="var(--text-muted)">{hint}</p>
	{/if}
</div>
