<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { _ } from 'svelte-i18n';
	import { formatAuthUserApiError } from '$lib/auth/api-errors';
	import { confirm } from '$lib/confirm';
	import { enterDemoMode, needsDemoReplaceConfirm } from '$lib/platform/demo-login';
	import { getServerProfile } from '$lib/platform/server-profile';
	import { setLocale } from '$lib/i18n';
	import { markSessionHint, persistLastUser, user } from '$lib/stores/auth';
	import { syncThemeFromUser } from '$lib/stores/theme';

	type Props = {
		onError: (message: string) => void;
		/** Parent action in flight (server-setup «Продолжить»). */
		blocked?: boolean;
		busy?: boolean;
	};

	let { onError, blocked = false, busy = $bindable(false) }: Props = $props();

	let confirming = $state(false);

	async function start() {
		if (busy || blocked || confirming) return;
		if (needsDemoReplaceConfirm(getServerProfile())) {
			confirming = true;
			const ok = await confirm({
				title: $_('androidAuth.demo.confirmTitle'),
				message: $_('androidAuth.demo.confirmMessage'),
				confirmLabel: $_('androidAuth.demo.confirmAction')
			});
			confirming = false;
			if (!ok || blocked) return;
		}
		busy = true;
		try {
			const signedIn = await enterDemoMode();
			user.set(signedIn);
			persistLastUser(signedIn);
			markSessionHint();
			setLocale(signedIn.language);
			syncThemeFromUser(signedIn.theme);
			await goto(resolve('/'));
		} catch (err) {
			onError(formatAuthUserApiError(err));
		} finally {
			busy = false;
		}
	}
</script>

<div class="mt-4">
	<button
		type="button"
		class="w-full text-center text-sm hover:underline disabled:cursor-not-allowed disabled:no-underline disabled:opacity-60"
		style:color="var(--text-muted)"
		disabled={busy || blocked || confirming}
		onclick={() => void start()}
	>
		{$_('androidAuth.demo.enter')}
	</button>
	<p class="mt-1 text-center text-xs" style:color="var(--text-muted)">
		{$_('androidAuth.demo.hint')}
	</p>
</div>
