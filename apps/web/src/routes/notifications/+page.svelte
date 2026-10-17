<script lang="ts">
	import { app } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { relative } from '#lib/time';
	import type { IconName } from '#lib/icons/paths';
	import AuthGate from '#lib/components/AuthGate.svelte';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import ListItem from '#lib/components/ListItem.svelte';

	const ICON: Record<string, IconName> = { reminder: 'schedule', team: 'groups', card: 'workspace_premium', broadcast: 'campaign', badge: 'military_tech' };
	let marked = false;
	$effect(() => {
		if (!marked && app.unread > 0) {
			marked = true;
			// Mark as read after the user had a moment to see what is new.
			setTimeout(() => void app.markAllRead().catch(() => {}), 1500);
		}
	});
</script>

<svelte:head><title>Powiadomienia · UniQuest</title></svelte:head>

<AuthGate>
	<TopAppBar title="Powiadomienia" back={href('/hero')} />
	<div class="page">
		{#if app.notifications.length === 0}
			<p class="body-medium muted">Nic nowego. Tu pojawią się odznaki, przypomnienia o zajęciach i wiadomości od drużyny.</p>
		{:else}
			<Card variant="outlined" padding={false}>
				{#each app.notifications as n (n.id)}
					<div class:unread={!n.readAt}>
						<ListItem icon={ICON[n.kind] ?? 'notifications'} overline={relative(n.createdAt)} headline={n.title} supporting={n.body} href={n.url ? href(n.url) : undefined} />
					</div>
				{/each}
			</Card>
		{/if}
	</div>
</AuthGate>

<style>
	.unread {
		background: color-mix(in srgb, var(--md-sys-color-primary-container) 50%, transparent);
	}
</style>
