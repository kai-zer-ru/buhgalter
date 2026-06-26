<script lang="ts">
	import { portal } from '$lib/actions/portal';
	import { dropdownListStyle } from '$lib/dropdown-position';
	import FieldHint from '$lib/components/FieldHint.svelte';
	import { _ } from 'svelte-i18n';
	import {
		buildDatetimeLocal,
		calendarCells,
		formatDateButtonLabel,
		formatDatetimeButtonLabel,
		parseDatetimeLocal
	} from '$lib/datetime-picker';

	type TimeMode = 'hidden' | 'optional' | 'visible';

	let {
		value = $bindable(''),
		id = 'datetime',
		label = '',
		hint = '',
		disabled = false,
		usePortal = false,
		timeMode = 'visible' as TimeMode,
		defaultTime = 'preserve' as 'now' | 'preserve' | string,
		required = false
	}: {
		value?: string;
		id?: string;
		label?: string;
		hint?: string;
		disabled?: boolean;
		usePortal?: boolean;
		timeMode?: TimeMode;
		defaultTime?: 'now' | 'preserve' | string;
		required?: boolean;
	} = $props();

	let open = $state(false);
	let triggerEl: HTMLButtonElement | undefined = $state();
	let panelEl: HTMLDivElement | undefined = $state();
	let panelStyle = $state('');
	let timeExpanded = $state(false);
	let timeValue = $state('');
	let viewYear = $state(new Date().getFullYear());
	let viewMonth = $state(new Date().getMonth() + 1);
	let panelView = $state<'days' | 'months' | 'years'>('days');
	let yearPageStart = $state(new Date().getFullYear() - 5);

	const monthShortNames = $derived(
		Array.from({ length: 12 }, (_, i) =>
			new Date(viewYear, i, 1).toLocaleDateString(undefined, { month: 'short' })
		)
	);

	const yearPageYears = $derived(Array.from({ length: 12 }, (_, i) => yearPageStart + i));

	const parsed = $derived(parseDatetimeLocal(value));

	const displayLabel = $derived(
		timeMode === 'visible' ? formatDatetimeButtonLabel(value) : formatDateButtonLabel(value)
	);

	const monthLabel = $derived(
		new Date(viewYear, viewMonth - 1, 1).toLocaleDateString(undefined, {
			month: 'long',
			year: 'numeric'
		})
	);

	const cells = $derived(calendarCells(viewYear, viewMonth));
	const weekdays = $derived([
		$_('datetime.weekday.mon'),
		$_('datetime.weekday.tue'),
		$_('datetime.weekday.wed'),
		$_('datetime.weekday.thu'),
		$_('datetime.weekday.fri'),
		$_('datetime.weekday.sat'),
		$_('datetime.weekday.sun')
	]);

	function syncViewFromValue() {
		const p = parseDatetimeLocal(value);
		if (p) {
			viewYear = p.year;
			viewMonth = p.month;
			timeValue = `${String(p.hour).padStart(2, '0')}:${String(p.minute).padStart(2, '0')}`;
		}
	}

	function effectiveTime(): { hour: number; minute: number } {
		if (timeMode === 'hidden') {
			return { hour: 0, minute: 0 };
		}
		if (timeMode === 'optional' && !timeExpanded) {
			if (defaultTime === 'now') {
				const now = new Date();
				return { hour: now.getHours(), minute: now.getMinutes() };
			}
			if (defaultTime !== 'preserve') {
				const [h, m] = defaultTime.split(':').map(Number);
				return { hour: h || 0, minute: m || 0 };
			}
			const p = parseDatetimeLocal(value);
			return { hour: p?.hour ?? 0, minute: p?.minute ?? 0 };
		}
		const [h, m] = (timeValue || '00:00').split(':').map(Number);
		return { hour: h || 0, minute: m || 0 };
	}

	function setDate(year: number, month: number, day: number) {
		viewYear = year;
		viewMonth = month;
		const { hour, minute } = effectiveTime();
		value = buildDatetimeLocal(year, month, day, hour, minute);
		if (timeMode !== 'visible') close();
	}

	function isSelectedCell(cell: { year: number; month: number; day: number }): boolean {
		return parsed?.day === cell.day && parsed?.month === cell.month && parsed?.year === cell.year;
	}

	function applyTime() {
		const p = parseDatetimeLocal(value);
		if (!p) return;
		const [h, m] = (timeValue || '00:00').split(':').map(Number);
		value = buildDatetimeLocal(p.year, p.month, p.day, h || 0, m || 0);
	}

	function prevMonth() {
		if (viewMonth === 1) {
			viewMonth = 12;
			viewYear -= 1;
		} else {
			viewMonth -= 1;
		}
	}

	function nextMonth() {
		if (viewMonth === 12) {
			viewMonth = 1;
			viewYear += 1;
		} else {
			viewMonth += 1;
		}
	}

	function openMonthsView() {
		panelView = 'months';
	}

	function openYearsView() {
		yearPageStart = viewYear - 5;
		panelView = 'years';
	}

	function selectYear(year: number) {
		viewYear = year;
		panelView = 'months';
	}

	function selectMonth(month: number) {
		viewMonth = month;
		panelView = 'days';
	}

	function prevYearPage() {
		yearPageStart -= 12;
	}

	function nextYearPage() {
		yearPageStart += 12;
	}

	function positionPanel() {
		if (!triggerEl) return;
		panelStyle = dropdownListStyle(triggerEl, 320, usePortal);
	}

	function openPanel() {
		if (disabled) return;
		syncViewFromValue();
		panelView = 'days';
		yearPageStart = viewYear - 5;
		open = true;
		requestAnimationFrame(positionPanel);
	}

	function close() {
		open = false;
		panelView = 'days';
	}

	function onDocumentPointerDown(event: PointerEvent) {
		const target = event.target as Node;
		if (triggerEl?.contains(target) || panelEl?.contains(target)) return;
		close();
	}

	$effect(() => {
		if (!open) return;
		document.addEventListener('pointerdown', onDocumentPointerDown, true);
		window.addEventListener('resize', positionPanel);
		window.addEventListener('scroll', positionPanel, true);
		return () => {
			document.removeEventListener('pointerdown', onDocumentPointerDown, true);
			window.removeEventListener('resize', positionPanel);
			window.removeEventListener('scroll', positionPanel, true);
		};
	});

	function onKeydown(event: KeyboardEvent) {
		if (event.key === 'Escape') {
			event.preventDefault();
			close();
		}
	}
</script>

<div class="relative">
	{#if label}
		<label class="mb-1.5 block text-sm font-medium" for={id}>{label}</label>
	{/if}
	<button
		{id}
		type="button"
		bind:this={triggerEl}
		class="input flex w-full cursor-pointer items-center justify-between gap-2 text-left"
		class:opacity-60={disabled}
		{disabled}
		aria-expanded={open}
		onclick={openPanel}
		onkeydown={onKeydown}
	>
		<span
			class="min-w-0 flex-1 truncate whitespace-nowrap"
			style:color={value ? 'var(--text)' : 'var(--text-muted)'}
		>
			{displayLabel || '—'}
		</span>
		<span aria-hidden="true" style:color="var(--text-muted)">📅</span>
	</button>

	{#if timeMode === 'optional'}
		<details bind:open={timeExpanded} class="mt-2">
			<summary class="cursor-pointer text-sm" style:color="var(--text-muted)">
				{$_('transactions.field.timeOptional')}
			</summary>
			<div class="mt-2">
				<input type="time" class="input w-full" bind:value={timeValue} onchange={applyTime} />
				<FieldHint text={$_('transactions.field.timeHint')} />
			</div>
		</details>
	{/if}

	{#if open}
		<div
			bind:this={panelEl}
			class="popover-panel w-72 p-3 {usePortal ? '' : 'absolute z-20'}"
			style={panelStyle}
			use:portal={usePortal ? document.body : null}
		>
			{#if panelView === 'days'}
				<div class="mb-2 flex items-center justify-between gap-2">
					<button type="button" class="btn-ghost px-2 py-1" onclick={prevMonth}>‹</button>
					<button
						type="button"
						class="text-sm font-medium capitalize hover:underline"
						onclick={openMonthsView}
					>
						{monthLabel}
					</button>
					<button type="button" class="btn-ghost px-2 py-1" onclick={nextMonth}>›</button>
				</div>
				<div
					class="mb-1 grid grid-cols-7 gap-1 text-center text-xs"
					style:color="var(--text-muted)"
				>
					{#each weekdays as weekday (weekday)}
						<span>{weekday}</span>
					{/each}
				</div>
				<div class="datetime-days-grid grid grid-cols-7 gap-1">
					{#each cells as cell, index (`${cell.year}-${cell.month}-${cell.day}-${index}`)}
						<button
							type="button"
							class="h-9 cursor-pointer rounded-lg px-0 py-2 text-sm transition hover:bg-[color-mix(in_srgb,var(--border)_45%,transparent)]"
							class:datetime-day-muted={!cell.inMonth}
							class:font-semibold={isSelectedCell(cell)}
							style:background-color={isSelectedCell(cell)
								? 'color-mix(in srgb, var(--primary) 28%, transparent)'
								: 'transparent'}
							onclick={() => setDate(cell.year, cell.month, cell.day)}
						>
							{cell.day}
						</button>
					{/each}
				</div>
			{:else if panelView === 'months'}
				<div class="mb-2 flex items-center justify-between gap-2">
					<button type="button" class="btn-ghost px-2 py-1" onclick={() => viewYear--}>‹</button>
					<button type="button" class="text-sm font-medium hover:underline" onclick={openYearsView}>
						{viewYear}
					</button>
					<button type="button" class="btn-ghost px-2 py-1" onclick={() => viewYear++}>›</button>
				</div>
				<div class="grid grid-cols-3 gap-2">
					{#each monthShortNames as name, index (index)}
						<button
							type="button"
							class="cursor-pointer rounded-lg px-2 py-2 text-sm capitalize transition hover:bg-[color-mix(in_srgb,var(--border)_45%,transparent)]"
							class:font-semibold={viewMonth === index + 1}
							style:background-color={viewMonth === index + 1
								? 'color-mix(in srgb, var(--primary) 28%, transparent)'
								: 'transparent'}
							onclick={() => selectMonth(index + 1)}
						>
							{name}
						</button>
					{/each}
				</div>
			{:else}
				<div class="mb-2 flex items-center justify-between gap-2">
					<button type="button" class="btn-ghost px-2 py-1" onclick={prevYearPage}>‹</button>
					<span class="text-sm font-medium">
						{yearPageYears[0]}–{yearPageYears[yearPageYears.length - 1]}
					</span>
					<button type="button" class="btn-ghost px-2 py-1" onclick={nextYearPage}>›</button>
				</div>
				<div class="grid grid-cols-3 gap-2">
					{#each yearPageYears as year (year)}
						<button
							type="button"
							class="cursor-pointer rounded-lg px-2 py-2 text-sm transition hover:bg-[color-mix(in_srgb,var(--border)_45%,transparent)]"
							class:font-semibold={viewYear === year}
							style:background-color={viewYear === year
								? 'color-mix(in srgb, var(--primary) 28%, transparent)'
								: 'transparent'}
							onclick={() => selectYear(year)}
						>
							{year}
						</button>
					{/each}
				</div>
			{/if}
			{#if timeMode === 'visible' && panelView === 'days'}
				<div class="mt-3 border-t pt-3" style:border-color="var(--border)">
					<label class="mb-1 block text-xs" style:color="var(--text-muted)" for="{id}-time">
						{$_('datetime.time')}
					</label>
					<input
						id="{id}-time"
						type="time"
						class="input w-full"
						bind:value={timeValue}
						onchange={applyTime}
					/>
				</div>
			{/if}
			<div class="mt-3 flex justify-end gap-2">
				<button type="button" class="btn-ghost text-sm" onclick={close}>
					{$_('datetime.done')}
				</button>
			</div>
		</div>
	{/if}
	{#if hint}
		<FieldHint text={hint} />
	{/if}
	<input type="hidden" {required} {value} />
</div>

<style>
	.datetime-day-muted {
		color: color-mix(in srgb, var(--text-muted) 72%, transparent);
	}

	.datetime-days-grid {
		grid-auto-rows: minmax(0, auto);
	}
</style>
