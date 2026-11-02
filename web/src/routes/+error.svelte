<script lang="ts">
	import { page } from '$app/state';
	import { m } from '#lib/copy/messages.ts';
</script>

<h1 class="caps">{page.status === 404 ? m.shell.notFound : m.shell.wentWrong}</h1>
<p>{page.error?.message}</p>
<p><a href="/">{m.shell.backHome}</a></p>

<style>
	h1 {
		font-size: var(--step-3);
		font-weight: 600;
		line-height: 1;
		margin-bottom: var(--space-4);
	}
	p {
		max-width: 65ch;
		margin-bottom: var(--space-3);
	}
</style>
