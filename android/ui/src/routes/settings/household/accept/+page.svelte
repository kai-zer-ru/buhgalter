<script lang="ts">
	import { page } from '$app/stores';
	import AcceptInviteForm from '$lib/household/AcceptInviteForm.svelte';

	const code = $derived($page.url.searchParams.get('code') ?? '');
</script>

<AcceptInviteForm initialCode={code} />
