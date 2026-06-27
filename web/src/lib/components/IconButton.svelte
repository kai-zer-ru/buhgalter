<script lang="ts">
	import type { Snippet } from 'svelte';
	import IconGlyph from '$lib/components/IconGlyph.svelte';

	export type IconName =
		| 'edit'
		| 'delete'
		| 'menu'
		| 'more-vertical'
		| 'logout'
		| 'add'
		| 'pay'
		| 'transfer'
		| 'save'
		| 'cancel'
		| 'create'
		| 'archive'
		| 'repeat'
		| 'bank';

	let {
		icon,
		label,
		variant = 'ghost',
		class: className = '',
		children,
		...rest
	}: {
		icon: IconName;
		label: string;
		variant?: 'ghost' | 'primary' | 'danger';
		class?: string;
		children?: Snippet;
		onclick?: (e: MouseEvent) => void;
		type?: 'button' | 'submit' | 'reset';
		disabled?: boolean;
		title?: string;
		'aria-expanded'?: boolean;
		'aria-pressed'?: boolean;
	} = $props();

	const btnClass = $derived(
		variant === 'primary'
			? 'btn-icon btn-primary'
			: variant === 'danger'
				? 'btn-icon btn-icon-danger'
				: 'btn-icon btn-ghost'
	);
</script>

<button
	type="button"
	class="{btnClass} {className}"
	aria-label={label}
	title={rest.title ?? label}
	{...rest}
>
	{#if children}
		{@render children()}
	{:else}
		<IconGlyph {icon} />
	{/if}
</button>
