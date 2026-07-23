<script lang="ts">
	import { portal } from '$lib/actions/portal';
	import { dropdownListStyle } from '$lib/dropdown-position';

	export type ComboboxOption = {
		value: string;
		label: string;
		disabled?: boolean;
	};

	let {
		value = $bindable(''),
		query = $bindable(''),
		options,
		id = 'combobox',
		label = '',
		hint = '',
		placeholder = '',
		emptyLabel = '',
		disabled = false,
		usePortal = false,
		filterOptions = true,
		onchange
	}: {
		value?: string;
		query?: string;
		options: ComboboxOption[];
		id?: string;
		label?: string;
		hint?: string;
		placeholder?: string;
		emptyLabel?: string;
		disabled?: boolean;
		usePortal?: boolean;
		filterOptions?: boolean;
		onchange?: (value: string) => void;
	} = $props();

	let open = $state(false);
	let inputEl: HTMLInputElement | undefined = $state();
	let listEl: HTMLUListElement | undefined = $state();
	let highlighted = $state(0);
	let listStyle = $state('');

	const listId = $derived(`${id}-list`);

	const visibleOptions = $derived.by(() => {
		const q = query.trim().toLowerCase();
		const list =
			filterOptions && q
				? options.filter((option) => option.label.toLowerCase().includes(q))
				: options;
		return list.filter((option) => !option.disabled || option.value === value);
	});

	$effect(() => {
		if (!open) {
			const selected = options.find((option) => option.value === value);
			query = selected?.label ?? value;
		}
	});

	function positionList() {
		if (!inputEl) return;
		const listHeight = Math.min(224, Math.max(visibleOptions.length, 1) * 40);
		listStyle = dropdownListStyle(inputEl, listHeight, usePortal);
	}

	function close() {
		open = false;
		const selected = options.find((option) => option.value === value);
		query = selected?.label ?? value;
	}

	function openList() {
		if (disabled) return;
		open = true;
		query = filterOptions ? '' : (options.find((option) => option.value === value)?.label ?? value);
		requestAnimationFrame(() => {
			positionList();
			const index = visibleOptions.findIndex((option) => option.value === value);
			highlighted = index >= 0 ? index : 0;
		});
	}

	function selectOption(next: string) {
		value = next;
		const selected = options.find((option) => option.value === next);
		query = selected?.label ?? next;
		onchange?.(next);
		close();
	}

	function onInput() {
		open = true;
		if (!filterOptions) return;
		const exact = options.find(
			(option) => option.label.toLowerCase() === query.trim().toLowerCase()
		);
		if (exact) value = exact.value;
		requestAnimationFrame(positionList);
	}

	function onBlur() {
		window.setTimeout(() => {
			if (!open) return;
			close();
		}, 150);
	}

	function onDocumentPointerDown(event: PointerEvent) {
		const target = event.target as Node;
		if (inputEl?.contains(target) || listEl?.contains(target)) return;
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

	function onKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			event.preventDefault();
			close();
			inputEl?.blur();
			return;
		}
		if (!open && (event.key === 'ArrowDown' || event.key === 'ArrowUp')) {
			event.preventDefault();
			openList();
			return;
		}
		if (!open) {
			if (event.key === 'Enter' && visibleOptions.length === 1) {
				event.preventDefault();
				selectOption(visibleOptions[0].value);
			}
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
		<input
			{id}
			bind:this={inputEl}
			class="input"
			type="text"
			role="combobox"
			aria-expanded={open}
			aria-controls={listId}
			aria-autocomplete="list"
			autocomplete="off"
			{placeholder}
			{disabled}
			bind:value={query}
			onfocus={openList}
			oninput={onInput}
			onblur={onBlur}
			onkeydown={onKeydown}
		/>
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
								class="w-full cursor-pointer px-4 py-2 text-left text-sm hover:opacity-90"
								class:font-medium={option.value === value}
								style:background-color={index === highlighted || option.value === value
									? 'color-mix(in srgb, var(--primary) 12%, transparent)'
									: 'transparent'}
								disabled={option.disabled}
								onmousedown={(event) => event.preventDefault()}
								onclick={() => selectOption(option.value)}
							>
								{option.label}
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
