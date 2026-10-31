<script lang="ts">
	import ToggleSwitch from '$lib/components/ToggleSwitch.svelte';
	import { onMount } from 'svelte';
	import { get } from 'svelte/store';
	import { _ } from 'svelte-i18n';
	import { MAX_HOME_SSIDS, normalizeHomeSsids } from '$lib/platform/server-profile';
	import { getCurrentWifiSsid } from '$lib/platform/wifi-subnet';

	type Props = {
		lanUrl: string;
		remoteUrl?: string;
		homeSsids?: string[];
		lanFallbackRemote?: boolean;
		/** `setup` — только LAN URL (первый запуск); `full` — внешний URL и Wi‑Fi (настройки) */
		variant?: 'setup' | 'full';
		showActiveStatus?: boolean;
		activeMode?: 'lan' | 'remote' | null;
		currentSsid?: string | null;
		/** External URL reported by the discovered server when it differs from the filled-in one. */
		remoteSuggestion?: string;
		onapplyremote?: () => void;
	};

	let {
		lanUrl = $bindable(),
		remoteUrl = $bindable(''),
		homeSsids = $bindable([]),
		lanFallbackRemote = $bindable(false),
		variant = 'full',
		showActiveStatus = false,
		activeMode = null,
		currentSsid = null,
		remoteSuggestion = '',
		onapplyremote
	}: Props = $props();

	let ssidHint = $state<string | null>(null);
	let permissionDenied = $state(false);
	let addingSsid = $state(false);

	const ssidSlots = $derived(
		Array.from({ length: MAX_HOME_SSIDS }, (_, index) => homeSsids[index] ?? '')
	);

	onMount(() => {
		void refreshCurrentSsid();
	});

	async function refreshCurrentSsid() {
		const result = await getCurrentWifiSsid();
		ssidHint = result.ssid;
		permissionDenied = Boolean(result.permissionDenied);
	}

	function updateSsidSlot(index: number, value: string) {
		const next = [...ssidSlots];
		next[index] = value;
		homeSsids = normalizeHomeSsids(next);
	}

	async function addCurrentSsid() {
		if (addingSsid) return;
		addingSsid = true;
		try {
			const result = await getCurrentWifiSsid({ requestPermission: true });
			ssidHint = result.ssid;
			permissionDenied = Boolean(result.permissionDenied);
			if (!result.ssid) return;
			const next = normalizeHomeSsids([...homeSsids, result.ssid]);
			homeSsids = next;
		} finally {
			addingSsid = false;
		}
	}

	function homeWifiPlaceholder(n: number): string {
		return get(_)('serverProfile.homeWifiPlaceholder', { values: { n } });
	}

	function detectedSsidLabel(ssid: string): string {
		return get(_)('serverProfile.detectedSsid', { values: { ssid } });
	}

	function currentSsidLabel(ssid: string): string {
		return get(_)('serverProfile.currentSsid', { values: { ssid } });
	}
</script>

{#if showActiveStatus}
	<div
		class="rounded-xl border px-3 py-2.5 text-sm"
		style:border-color="var(--border)"
		style:background-color="color-mix(in srgb, var(--primary) 6%, var(--bg-elevated))"
	>
		<p class="font-medium">
			{activeMode === 'lan'
				? $_('serverProfile.activeLan')
				: activeMode === 'remote'
					? $_('serverProfile.activeRemote')
					: $_('serverProfile.activeUnknown')}
		</p>
		<p class="mt-0.5 text-xs" style:color="var(--text-muted)">
			{#if currentSsid}
				{currentSsidLabel(currentSsid)}
			{:else if permissionDenied}
				{$_('serverProfile.ssidPermission')}
			{:else}
				{$_('serverProfile.currentSsidUnknown')}
			{/if}
		</p>
	</div>
{/if}

<div class="space-y-4">
	<div>
		<label class="field-label" for="server-lan-url">
			{$_('serverProfile.lanUrlLabel')}
		</label>
		<input
			id="server-lan-url"
			class="input"
			type="url"
			inputmode="url"
			placeholder="http://192.168.1.10:8765"
			bind:value={lanUrl}
			required
			autocomplete="url"
		/>
		<p class="mt-1.5 text-xs" style:color="var(--text-muted)">{$_('serverSetup.portHint')}</p>
	</div>

	{#if variant === 'full'}
		<div>
			<label class="field-label" for="server-remote-url">
				{$_('serverProfile.remoteUrlLabel')}
			</label>
			<input
				id="server-remote-url"
				class="input"
				type="url"
				inputmode="url"
				placeholder="https://buh.example.com"
				bind:value={remoteUrl}
				autocomplete="url"
			/>
			<p class="mt-1.5 text-xs" style:color="var(--text-muted)">
				{$_('serverProfile.remoteUrlHint')}
			</p>
			{#if remoteSuggestion}
				<div class="mt-2 flex items-center justify-between gap-3 text-xs">
					<span class="min-w-0 break-all" style:color="var(--text-muted)">
						{$_('serverProfile.remoteSuggested', { values: { url: remoteSuggestion } })}
					</span>
					<button
						type="button"
						class="btn-ghost shrink-0 text-xs"
						onclick={() => onapplyremote?.()}
					>
						{$_('serverProfile.remoteSuggestedApply')}
					</button>
				</div>
			{/if}
			{#if remoteUrl.trim()}
				<div class="mt-3">
					<div class="flex items-center justify-between gap-4">
						<span class="min-w-0 text-sm">
							<span class="font-medium">{$_('serverProfile.lanFallbackRemoteLabel')}</span>
							<span class="mt-0.5 block text-xs" style:color="var(--text-muted)"
								>{$_('serverProfile.lanFallbackRemoteHint')}</span
							>
						</span>
						<ToggleSwitch
							checked={lanFallbackRemote}
							label={$_('serverProfile.lanFallbackRemoteLabel')}
							onchange={() => (lanFallbackRemote = !lanFallbackRemote)}
						/>
					</div>
				</div>
			{/if}
		</div>

		<fieldset class="space-y-3">
			<legend class="field-label">{$_('serverProfile.homeWifiTitle')}</legend>
			<p class="text-xs" style:color="var(--text-muted)">{$_('serverProfile.homeWifiHint')}</p>
			{#each ssidSlots as ssid, index (index)}
				<input
					class="input"
					type="text"
					placeholder={homeWifiPlaceholder(index + 1)}
					value={ssid}
					oninput={(event) => updateSsidSlot(index, event.currentTarget.value)}
					autocomplete="off"
					spellcheck={false}
				/>
			{/each}
			<div class="flex flex-wrap items-center gap-2">
				<button
					type="button"
					class="btn-ghost text-sm"
					disabled={addingSsid || homeSsids.length >= MAX_HOME_SSIDS}
					onclick={() => void addCurrentSsid()}
				>
					{addingSsid ? $_('common.loading') : $_('serverProfile.addCurrentWifi')}
				</button>
				{#if ssidHint}
					<span class="text-xs" style:color="var(--text-muted)">
						{detectedSsidLabel(ssidHint)}
					</span>
				{/if}
			</div>
			{#if permissionDenied}
				<p class="text-xs" style:color="var(--warning)">{$_('serverProfile.ssidPermission')}</p>
			{/if}
		</fieldset>
	{/if}
</div>
