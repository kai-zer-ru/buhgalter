<script lang="ts">
	import { onMount } from 'svelte';
	import { _, locale } from 'svelte-i18n';
	import { tr } from '$lib/i18n';
	import Combobox from '$lib/components/Combobox.svelte';
	import { TIMEZONE_FALLBACK } from '$lib/timezones';

	let { value = $bindable('UTC'), id = 'timezone', label = 'Часовой пояс', hint = '' } = $props();

	let allTimezones = $state<string[]>([...TIMEZONE_FALLBACK]);
	let query = $state(value);

	const options = $derived(
		allTimezones.map((tz) => ({
			value: tz,
			label: tz
		}))
	);

	const hintText = $derived.by(() => {
		void $locale;
		return `${hint || tr('settings.timezone.hint')} (${allTimezones.length} ${tr('settings.timezone.zones')})`;
	});

	onMount(() => {
		if (typeof Intl !== 'undefined' && 'supportedValuesOf' in Intl) {
			try {
				allTimezones = [...Intl.supportedValuesOf('timeZone')].sort();
			} catch {
				allTimezones = [...TIMEZONE_FALLBACK];
			}
		}
		if (value && !allTimezones.includes(value)) {
			allTimezones = [value, ...allTimezones].sort();
		}
	});

	function onchange(next: string) {
		value = next;
		query = next;
	}
</script>

<Combobox
	{id}
	{label}
	hint={hintText}
	placeholder="Europe/Moscow"
	bind:value
	bind:query
	{options}
	emptyLabel={$_('common.notFound')}
	{onchange}
/>
