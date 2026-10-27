<script lang="ts">
	import { page } from '$app/stores';
	import AcceptInviteForm from '$lib/household/AcceptInviteForm.svelte';

	const code = $derived($page.params.code ?? '');
</script>

<AcceptInviteForm initialCode={code} showOpenInApp />
