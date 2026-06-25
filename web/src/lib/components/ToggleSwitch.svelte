<script lang="ts">
	type Props = {
		checked: boolean;
		disabled?: boolean;
		label: string;
		onchange?: () => void;
	};

	let { checked, disabled = false, label, onchange }: Props = $props();
</script>

<button
	type="button"
	role="switch"
	aria-label={label}
	aria-checked={checked}
	{disabled}
	class="relative h-6 w-11 shrink-0 cursor-pointer rounded-full transition-colors disabled:cursor-not-allowed disabled:opacity-45"
	style:background-color={checked ? 'var(--primary)' : 'var(--border)'}
	onclick={() => {
		if (!disabled) onchange?.();
	}}
>
	<span
		class="absolute top-0.5 left-0.5 h-5 w-5 rounded-full bg-white shadow transition-transform"
		class:translate-x-5={checked}
	></span>
</button>
