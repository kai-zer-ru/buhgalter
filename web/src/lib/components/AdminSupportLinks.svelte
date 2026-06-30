<script lang="ts">
	import { _ } from 'svelte-i18n';

	const REPOSITORY_URL = 'https://github.com/kai-zer-ru/buhgalter';
	const SUPPORT_URL = 'https://www.tbank.ru/rm/r_wKLcbFgjYa.ncgWMwrHSA/vyQvd5941/';
</script>

<div class="card max-w-lg !p-3 sm:!p-3">
	<div class="flex flex-wrap gap-2">
		<a
			href={SUPPORT_URL}
			target="_blank"
			rel="noopener noreferrer"
			class="btn-primary inline-flex items-center gap-2"
		>
			<svg
				aria-hidden="true"
				class="h-4 w-4 shrink-0"
				viewBox="0 0 24 24"
				fill="none"
				stroke="currentColor"
				stroke-width="2"
				stroke-linecap="round"
				stroke-linejoin="round"
			>
				<path
					d="M20.84 4.61a5.5 5.5 0 0 0-7.78 0L12 5.67l-1.06-1.06a5.5 5.5 0 0 0-7.78 7.78l1.06 1.06L12 21.23l7.78-7.78 1.06-1.06a5.5 5.5 0 0 0 0-7.78z"
				/>
			</svg>
			{$_('admin.support.author')}
		</a>
		<a
			href={REPOSITORY_URL}
			target="_blank"
			rel="noopener noreferrer"
			class="btn-ghost inline-flex items-center gap-2"
		>
			<svg aria-hidden="true" class="h-4 w-4 shrink-0" viewBox="0 0 24 24" fill="currentColor">
				<path
					d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.113.82-.258.82-.577 0-.285-.01-1.04-.015-2.04-3.338.724-4.042-1.61-4.042-1.61-.546-1.385-1.335-1.755-1.335-1.755-1.087-.744.084-.729.084-.729 1.205.084 1.838 1.236 1.838 1.236 1.07 1.835 2.809 1.305 3.495.998.108-.776.417-1.305.76-1.604-2.665-.305-5.467-1.334-5.467-5.931 0-1.311.469-2.381 1.236-3.221-.124-.303-.535-1.524.117-3.176 0 0 1.008-.322 3.301 1.23.957-.266 1.983-.399 3.003-.404 1.02.005 2.047.138 3.006.404 2.291-1.552 3.297-1.23 3.297-1.23.653 1.653.242 2.874.118 3.176.77.84 1.235 1.911 1.235 3.221 0 4.609-2.807 5.624-5.479 5.921.43.372.823 1.102.823 2.222v3.293c0 .319.192.694.801.576 4.765-1.589 8.199-6.086 8.199-11.386 0-6.627-5.373-12-12-12z"
				/>
			</svg>
			{$_('admin.support.repository')}
		</a>
	</div>
</div>
