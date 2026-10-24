<script lang="ts">
	import { _ } from 'svelte-i18n';
	import type { NotificationInlineKeyboard } from '$lib/api/client';
	import ReorderDragGhost from '$lib/components/ReorderDragGhost.svelte';
	import Select from '$lib/components/Select.svelte';
	import { beginPointerDrag, type DragGhostView } from '$lib/drag-reorder';
	import {
		INBOUND_ONLY_TYPES,
		addButtonToRow,
		addRow,
		buttonTypesFor,
		composeTemplateRows,
		createLinkPreset,
		linkPresetsForTrigger,
		moveButton,
		moveRowByIds,
		normalizeInlineKeyboard,
		removeButton,
		removeRow,
		splitTemplateKeyboard,
		toApiKeyboard,
		toEditableKeyboard,
		updateButton,
		type EditableKeyboard,
		type EditableKeyboardButton,
		type KeyboardLinkPreset,
		type KeyboardProvider
	} from '$lib/settings/keyboard';

	type Props = {
		provider: KeyboardProvider;
		keyboard?: NotificationInlineKeyboard;
		disabled?: boolean;
		/** Stable prefix for data-testid / dragKind isolation across builders on one page. */
		scope?: string;
		/** Template trigger_type; null/undefined → global multi-presets. */
		triggerType?: string | null;
		/**
		 * When set, this builder is a template: global buttons of the channel are shown
		 * inside the layout (locked). Omit on the global keyboard editor.
		 */
		includeGlobal?: NotificationInlineKeyboard | null;
		onchange?: (keyboard: NotificationInlineKeyboard) => void;
	};

	let {
		provider,
		keyboard = $bindable(normalizeInlineKeyboard(undefined)),
		disabled = false,
		scope = 'kb',
		triggerType = null,
		includeGlobal = undefined,
		onchange
	}: Props = $props();

	function visualFromProps(): EditableKeyboard {
		if (includeGlobal !== undefined) {
			return { rows: composeTemplateRows(includeGlobal, keyboard) };
		}
		return toEditableKeyboard(keyboard);
	}

	function externalKey(): string {
		if (includeGlobal !== undefined) {
			return JSON.stringify({
				t: normalizeInlineKeyboard(keyboard),
				g: normalizeInlineKeyboard(includeGlobal)
			});
		}
		return JSON.stringify(normalizeInlineKeyboard(keyboard));
	}

	let editable = $state<EditableKeyboard>(visualFromProps());
	let lastExternalJson = $state(externalKey());
	let dragGhost = $state<DragGhostView | null>(null);
	let draggingId = $state<string | null>(null);
	let overId = $state<string | null>(null);

	const rowIds = $derived(editable.rows.map((row, i) => `row-${scope}-${i}-${row[0]?.id ?? i}`));
	const typeOptions = $derived(
		buttonTypesFor(provider).map((t) => ({
			value: t,
			label: $_(`settings.notifications.keyboard.type.${t}`)
		}))
	);
	const linkPresets = $derived(linkPresetsForTrigger(triggerType));

	$effect(() => {
		const incoming = externalKey();
		if (incoming !== lastExternalJson) {
			lastExternalJson = incoming;
			editable = visualFromProps();
		}
	});

	function commit(next: EditableKeyboard) {
		editable = next;
		const api =
			includeGlobal !== undefined ? splitTemplateKeyboard(next.rows) : toApiKeyboard(next);
		lastExternalJson = externalKeyFrom(api);
		keyboard = api;
		onchange?.(api);
	}

	function externalKeyFrom(api: NotificationInlineKeyboard): string {
		if (includeGlobal !== undefined) {
			return JSON.stringify({
				t: normalizeInlineKeyboard(api),
				g: normalizeInlineKeyboard(includeGlobal)
			});
		}
		return JSON.stringify(normalizeInlineKeyboard(api));
	}

	function dragBindings(onDrop: (from: string, to: string) => void) {
		return {
			isDisabled: () => disabled,
			setGhost: (g: DragGhostView | null) => (dragGhost = g),
			setDraggingId: (id: string | null) => (draggingId = id),
			setOverId: (id: string | null) => (overId = id),
			onDrop
		};
	}

	function startRowDrag(e: PointerEvent, rowId: string, rowEl: HTMLElement) {
		beginPointerDrag({
			e,
			id: rowId,
			rowEl,
			dragKind: `${scope}-row`,
			...dragBindings((from, to) => {
				const nextRows = moveRowByIds(editable.rows, rowIds, from, to);
				if (nextRows) commit({ rows: nextRows });
			})
		});
	}

	function startButtonDrag(e: PointerEvent, btnId: string, rowEl: HTMLElement) {
		beginPointerDrag({
			e,
			id: btnId,
			rowEl,
			dragKind: `${scope}-btn`,
			...dragBindings((from, to) => {
				const nextRows = moveButton(editable.rows, from, to);
				if (nextRows) commit({ rows: nextRows });
			})
		});
	}

	function onAddRow() {
		commit({ rows: addRow(editable.rows, provider) });
	}

	function onAddButton(rowIndex: number) {
		const next = addButtonToRow(editable.rows, rowIndex, provider);
		if (next) commit({ rows: next });
	}

	function rowHasLocked(rowIndex: number) {
		return editable.rows[rowIndex]?.some((btn) => btn.locked) ?? false;
	}

	function onRemoveRow(rowIndex: number) {
		if (rowHasLocked(rowIndex)) return;
		commit({ rows: removeRow(editable.rows, rowIndex) });
	}

	function onRemoveButton(buttonId: string) {
		const locked = editable.rows.some((row) =>
			row.some((btn) => btn.id === buttonId && btn.locked)
		);
		if (locked) return;
		commit({ rows: removeButton(editable.rows, buttonId) });
	}

	function onPatchButton(buttonId: string, patch: Partial<EditableKeyboardButton>) {
		const locked = editable.rows.some((row) =>
			row.some((btn) => btn.id === buttonId && btn.locked)
		);
		if (locked) return;
		commit({ rows: updateButton(editable.rows, buttonId, patch) });
	}

	function onPresetLink(def: KeyboardLinkPreset) {
		const preset = createLinkPreset(provider, def);
		preset.text = $_(def.labelKey);
		commit({ rows: [...editable.rows, [preset]] });
	}

	function fieldLabel(key: string) {
		return $_(`settings.notifications.keyboard.field.${key}`);
	}
</script>

{#if dragGhost}
	<ReorderDragGhost ghost={dragGhost} />
{/if}

<div
	class="space-y-3"
	data-testid={`keyboard-builder-${scope}`}
	class:opacity-60={disabled}
	class:pointer-events-none={disabled}
>
	<div class="flex flex-wrap items-center gap-2">
		<button type="button" class="btn-ghost" {disabled} onclick={onAddRow}>
			{$_('settings.notifications.keyboard.add_row')}
		</button>
		{#each linkPresets as def (def.url)}
			<button type="button" class="btn-ghost" {disabled} onclick={() => onPresetLink(def)}>
				{$_(def.chipKey)}
			</button>
		{/each}
	</div>

	{#if editable.rows.length === 0}
		<p class="text-sm" style:color="var(--text-muted)">
			{$_('settings.notifications.keyboard.empty')}
		</p>
	{/if}

	<div class="space-y-3">
		{#each editable.rows as row, rowIndex (rowIds[rowIndex])}
			<div
				class="rounded border p-3 space-y-2 transition-opacity"
				style:border-color="var(--border)"
				class:opacity-30={draggingId === rowIds[rowIndex]}
				data-drag-id={rowIds[rowIndex]}
				data-drag-kind={`${scope}-row`}
			>
				<div class="flex flex-wrap items-center gap-2" data-drag-row>
					<span
						class="btn-icon btn-ghost cursor-grab touch-none text-lg leading-none select-none active:cursor-grabbing"
						role="button"
						tabindex="-1"
						aria-label={$_('settings.notifications.keyboard.drag_row')}
						onpointerdown={(e) =>
							startRowDrag(
								e,
								rowIds[rowIndex],
								e.currentTarget.closest('[data-drag-id]') as HTMLElement
							)}
					>
						⠿
					</span>
					<span class="text-xs font-medium" style:color="var(--text-muted)">
						{$_('settings.notifications.keyboard.row_label', { values: { n: rowIndex + 1 } })}
					</span>
					<button
						type="button"
						class="btn-ghost ml-auto"
						{disabled}
						onclick={() => onAddButton(rowIndex)}
					>
						{$_('settings.notifications.keyboard.add_button')}
					</button>
					{#if !rowHasLocked(rowIndex)}
						<button
							type="button"
							class="btn-ghost"
							{disabled}
							onclick={() => onRemoveRow(rowIndex)}
						>
							{$_('settings.notifications.keyboard.remove_row')}
						</button>
					{/if}
				</div>

				<div class="flex flex-wrap gap-2">
					{#each row as btn (btn.id)}
						{@const fieldDisabled = disabled || !!btn.locked}
						<div
							class="min-w-[14rem] flex-1 rounded border p-2 space-y-2 transition-opacity"
							style:border-color={overId === btn.id ? 'var(--primary)' : 'var(--border)'}
							class:opacity-30={draggingId === btn.id}
							data-drag-id={btn.id}
							data-drag-kind={`${scope}-btn`}
							data-testid={`keyboard-btn-${btn.id}`}
							data-locked={btn.locked ? 'true' : 'false'}
						>
							<div class="flex items-center gap-1">
								<span
									class="btn-icon btn-ghost cursor-grab touch-none select-none active:cursor-grabbing"
									role="button"
									tabindex="-1"
									aria-label={$_('settings.notifications.keyboard.drag_button')}
									onpointerdown={(e) =>
										startButtonDrag(
											e,
											btn.id,
											e.currentTarget.closest('[data-drag-id]') as HTMLElement
										)}
								>
									⠿
								</span>
								{#if btn.locked}
									<span class="ml-auto text-xs" style:color="var(--text-muted)">
										{$_('settings.notifications.keyboard.locked_global')}
									</span>
								{:else}
									<button
										type="button"
										class="btn-ghost ml-auto text-xs"
										{disabled}
										onclick={() => onRemoveButton(btn.id)}
									>
										{$_('settings.notifications.keyboard.remove_button')}
									</button>
								{/if}
							</div>

							<label class="block space-y-1 text-xs">
								<span>{fieldLabel('type')}</span>
								<Select
									id={`${scope}-type-${btn.id}`}
									value={btn.type}
									options={typeOptions}
									disabled={disabled || !!btn.locked}
									onchange={(v) => onPatchButton(btn.id, { type: v })}
								/>
							</label>

							<label class="block space-y-1 text-xs">
								<span>{fieldLabel('text')}</span>
								<input
									class="input w-full"
									type="text"
									value={btn.text}
									disabled={fieldDisabled}
									oninput={(e) =>
										onPatchButton(btn.id, {
											text: (e.currentTarget as HTMLInputElement).value
										})}
								/>
							</label>

							{#if btn.type === 'url' || btn.type === 'link' || btn.type === 'open_app'}
								<label class="block space-y-1 text-xs">
									<span>{fieldLabel('url')}</span>
									<input
										class="input w-full"
										type="text"
										value={btn.url ?? ''}
										disabled={fieldDisabled}
										oninput={(e) =>
											onPatchButton(btn.id, {
												url: (e.currentTarget as HTMLInputElement).value
											})}
									/>
								</label>
							{:else if btn.type === 'callback_data'}
								<label class="block space-y-1 text-xs">
									<span>{fieldLabel('callback_data')}</span>
									<input
										class="input w-full"
										type="text"
										value={btn.callback_data ?? ''}
										disabled={fieldDisabled}
										oninput={(e) =>
											onPatchButton(btn.id, {
												callback_data: (e.currentTarget as HTMLInputElement).value
											})}
									/>
								</label>
							{:else if btn.type === 'web_app'}
								<label class="block space-y-1 text-xs">
									<span>{fieldLabel('web_app_url')}</span>
									<input
										class="input w-full"
										type="text"
										value={btn.web_app?.url ?? ''}
										disabled={fieldDisabled}
										oninput={(e) =>
											onPatchButton(btn.id, {
												web_app: { url: (e.currentTarget as HTMLInputElement).value }
											})}
									/>
								</label>
							{:else if btn.type === 'login_url'}
								<label class="block space-y-1 text-xs">
									<span>{fieldLabel('login_url')}</span>
									<input
										class="input w-full"
										type="text"
										value={btn.login_url?.url ?? ''}
										disabled={fieldDisabled}
										oninput={(e) =>
											onPatchButton(btn.id, {
												login_url: { url: (e.currentTarget as HTMLInputElement).value }
											})}
									/>
								</label>
							{:else if btn.type === 'copy_text'}
								<label class="block space-y-1 text-xs">
									<span>{fieldLabel('copy_text')}</span>
									<input
										class="input w-full"
										type="text"
										value={btn.copy_text?.text ?? ''}
										disabled={fieldDisabled}
										oninput={(e) =>
											onPatchButton(btn.id, {
												copy_text: { text: (e.currentTarget as HTMLInputElement).value }
											})}
									/>
								</label>
							{:else if btn.type === 'callback' || btn.type === 'clipboard' || btn.type === 'message'}
								<label class="block space-y-1 text-xs">
									<span>{fieldLabel('payload')}</span>
									<input
										class="input w-full"
										type="text"
										value={btn.payload ?? ''}
										disabled={fieldDisabled}
										oninput={(e) =>
											onPatchButton(btn.id, {
												payload: (e.currentTarget as HTMLInputElement).value
											})}
									/>
								</label>
							{:else if btn.type === 'switch_inline_query'}
								<label class="block space-y-1 text-xs">
									<span>{fieldLabel('switch_inline_query')}</span>
									<input
										class="input w-full"
										type="text"
										value={btn.switch_inline_query ?? ''}
										disabled={fieldDisabled}
										oninput={(e) =>
											onPatchButton(btn.id, {
												switch_inline_query: (e.currentTarget as HTMLInputElement).value
											})}
									/>
								</label>
							{:else if btn.type === 'switch_inline_query_current_chat'}
								<label class="block space-y-1 text-xs">
									<span>{fieldLabel('switch_inline_query')}</span>
									<input
										class="input w-full"
										type="text"
										value={btn.switch_inline_query_current_chat ?? ''}
										disabled={fieldDisabled}
										oninput={(e) =>
											onPatchButton(btn.id, {
												switch_inline_query_current_chat: (e.currentTarget as HTMLInputElement)
													.value
											})}
									/>
								</label>
							{/if}

							{#if INBOUND_ONLY_TYPES.has(btn.type)}
								<p class="text-xs" style:color="var(--text-muted)">
									{$_('settings.notifications.keyboard.inbound_hint')}
								</p>
							{/if}
						</div>
					{/each}
				</div>
			</div>
		{/each}
	</div>

	{#if editable.rows.some((r) => r.length > 0)}
		<div class="space-y-1">
			<p class="text-xs font-medium" style:color="var(--text-muted)">
				{$_('settings.notifications.keyboard.preview_grid')}
			</p>
			<div
				class="inline-flex flex-col gap-1 rounded border p-2"
				style:border-color="var(--border)"
				data-testid={`keyboard-preview-${scope}`}
			>
				{#each editable.rows as row, ri (rowIds[ri])}
					<div class="flex flex-wrap gap-1">
						{#each row as btn (btn.id)}
							<span
								class="rounded px-2 py-1 text-xs"
								style:background="color-mix(in srgb, var(--primary) 12%, transparent)"
								style:color="var(--text)"
							>
								{btn.text.trim() || $_('settings.notifications.keyboard.preview_untitled')}
							</span>
						{/each}
					</div>
				{/each}
			</div>
		</div>
	{/if}
</div>
