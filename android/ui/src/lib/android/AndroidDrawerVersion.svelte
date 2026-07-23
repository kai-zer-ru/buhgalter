<script lang="ts">
	import { _ } from 'svelte-i18n';
	import type { AppVersionInfo } from '$lib/version-check';

	type Props = {
		info: AppVersionInfo;
		onshowUpdate: () => void;
	};

	let { info, onshowUpdate }: Props = $props();

	const displayVersion = $derived(`v${info.appVersion.replace(/^v/i, '')}`);
</script>

<div class="android-drawer-version">
	<button
		type="button"
		class="android-drawer-version-btn"
		aria-label={$_('nav.appVersion', { values: { version: displayVersion } })}
		onclick={onshowUpdate}
	>
		<span class="android-drawer-version-label">{displayVersion}</span>
		{#if info.versionMismatch}
			<span class="android-drawer-version-warn" aria-hidden="true">!</span>
		{/if}
	</button>
</div>
