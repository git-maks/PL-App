<script lang="ts">
	import { onDestroy } from 'svelte';
	import { fly } from 'svelte/transition';
	import { flip } from 'svelte/animate';
	import { app, errorMessage } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { toast } from '#lib/state/toast.svelte';
	import type { IconName } from '#lib/icons/paths';
	import type { Leaderboard, TeamInvite } from '#lib/api';
	import AuthGate from '#lib/components/AuthGate.svelte';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import Button from '#lib/components/Button.svelte';
	import TextField from '#lib/components/TextField.svelte';
	import Dialog from '#lib/components/Dialog.svelte';
	import Icon from '#lib/components/Icon.svelte';
	import IconButton from '#lib/components/IconButton.svelte';
	import PixelAvatar from '#lib/components/PixelAvatar.svelte';
	import Chip from '#lib/components/Chip.svelte';
	import QrCode from '#lib/qr/QrCode.svelte';

	const EMBLEMS: IconName[] = ['shield', 'star', 'science', 'lightbulb', 'sports_soccer', 'casino', 'public', 'military_tech'];

	let name = $state('');
	let emblem = $state<IconName>('shield');
	let code = $state('');
	let busy = $state(false);
	let error = $state<string | null>(null);
	let invite = $state<TeamInvite | null>(null);
	let inviteOpen = $state(false);
	let leaveOpen = $state(false);
	let board = $state<Leaderboard | null>(null);
	let boardType = $state<'team' | 'solo'>('team');
	let now = $state(Date.now());
	const tick = setInterval(() => (now = Date.now()), 1000);
	onDestroy(() => clearInterval(tick));

	const team = $derived(app.team);
	const isCaptain = $derived(!!team && team.captainId === app.me?.id);
	const left = $derived(invite ? Math.max(0, Math.floor((new Date(invite.expiresAt).getTime() - now) / 1000)) : 0);

	async function loadBoard() {
		if (!app.event) return;
		try {
			board = await app.api.leaderboard(app.event.id, boardType);
		} catch {
			board = null;
		}
	}
	$effect(() => {
		void boardType;
		void team?.score;
		if (app.me) void loadBoard();
	});

	async function create(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = null;
		try {
			await app.createTeam(name, emblem);
			toast('Drużyna utworzona! Zaproś znajomych kodem QR.');
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}
	async function join(e: SubmitEvent) {
		e.preventDefault();
		busy = true;
		error = null;
		try {
			await app.joinTeam(code);
			toast('Witaj w drużynie!');
		} catch (err) {
			error = errorMessage(err);
		} finally {
			busy = false;
		}
	}
	async function makeInvite() {
		if (!team) return;
		try {
			invite = await app.api.createInvite(team.id);
			inviteOpen = true;
		} catch (err) {
			toast(errorMessage(err));
		}
	}
	async function shareInvite() {
		if (!invite) return;
		const data = { title: 'Dołącz do mojej drużyny w UniQuest', url: invite.url };
		if (navigator.share) await navigator.share(data).catch(() => {});
		else {
			await navigator.clipboard?.writeText(invite.url);
			toast('Link skopiowany do schowka.');
		}
	}
	async function leave() {
		leaveOpen = false;
		await app.leaveTeam().catch((e) => toast(errorMessage(e)));
	}
	async function kick(userId: string) {
		if (!team) return;
		await app.api.removeMember(team.id, userId).catch((e) => toast(errorMessage(e)));
		await app.refresh();
	}
</script>

<svelte:head><title>Drużyna · UniQuest</title></svelte:head>

<AuthGate reason="Drużyny są dostępne dla bohaterów.">
	<TopAppBar title="Drużyna" />
	<div class="page">
		{#if !app.event}
			<Card variant="tertiary"><p class="body-medium">Drużyny tworzy się podczas wydarzenia UniQuest.</p></Card>
		{:else if !team}
			<p class="body-medium muted">
				Niektóre zadania są drużynowe — gdy dotrze na nie co najmniej {app.event.teamMin} osoby z drużyny, każdy dostaje bonus ×{app.event.config.teamBonus.toLocaleString('pl-PL')}. Drużyna liczy do {app.event.teamMax} osób.
			</p>
			<Card variant="filled">
				<form class="stack" onsubmit={create}>
					<h2 class="title-medium">Załóż drużynę</h2>
					<TextField label="Nazwa drużyny" bind:value={name} maxlength={24} autocomplete="off" />
					<div class="emblems" role="radiogroup" aria-label="Emblemat drużyny">
						{#each EMBLEMS as em (em)}
							<button type="button" role="radio" aria-checked={emblem === em} aria-label={em} class="em interactive" class:on={emblem === em} onclick={() => (emblem = em)}>
								<Icon name={em} />
							</button>
						{/each}
					</div>
					<Button type="submit" icon="add" loading={busy} disabled={name.trim().length < 2}>Utwórz drużynę</Button>
				</form>
			</Card>
			<Card variant="outlined">
				<form class="stack" onsubmit={join}>
					<h2 class="title-medium">Dołącz do drużyny</h2>
					<p class="body-medium muted">Zeskanuj kod QR z telefonu kapitana albo wpisz kod zaproszenia.</p>
					<Button variant="tonal" icon="qr_code_scanner" href={href('/scan')}>Skanuj kod zaproszenia</Button>
					<TextField label="Kod zaproszenia" bind:value={code} maxlength={64} autocomplete="off" />
					<Button type="submit" variant="outlined" icon="group_add" loading={busy} disabled={code.trim().length < 6}>Dołącz</Button>
				</form>
			</Card>
			{#if error}<p class="error body-medium" role="alert">{error}</p>{/if}
		{:else}
			<Card variant="primary">
				<div class="team-head">
					<span class="emblem"><Icon name={(team.emblem as IconName) ?? 'shield'} size={32} /></span>
					<div class="grow">
						<h2 class="title-large">{team.name}</h2>
						<p class="body-small">{team.members.length}/{team.maxMembers} członków · <span class="pixel">{Math.round(team.score)} pkt</span></p>
					</div>
				</div>
			</Card>

			<section class="stack" aria-labelledby="members-h">
				<div class="row">
					<h2 id="members-h" class="section-title">Członkowie</h2>
					<span class="spacer"></span>
					{#if isCaptain && team.members.length < team.maxMembers}
						<Button variant="tonal" icon="person_add" onclick={makeInvite}>Zaproś</Button>
					{/if}
				</div>
				<ul class="members" aria-live="polite">
					{#each team.members as m (m.userId)}
						<li animate:flip={{ duration: 300 }} in:fly={{ y: 16, duration: 350 }}>
							<PixelAvatar src={m.avatarUrl} seed={m.userId} size={44} />
							<div class="grow">
								<p class="title-small">{m.nickname}{m.userId === app.me?.id ? ' (Ty)' : ''}</p>
								<p class="body-small muted">{m.role === 'captain' ? 'Kapitan · ' : ''}{Math.round(m.xp)} XP</p>
							</div>
							{#if m.role === 'captain'}<Icon name="military_tech" label="Kapitan" />{/if}
							{#if isCaptain && m.userId !== app.me?.id}
								<IconButton icon="close" label="Usuń {m.nickname} z drużyny" onclick={() => kick(m.userId)} />
							{/if}
						</li>
					{/each}
				</ul>
				{#if team.members.length < (app.event?.teamMin ?? 2)}
					<p class="body-small muted">Zaproś co najmniej jedną osobę, aby zdobywać bonusy na zadaniach drużynowych.</p>
				{/if}
			</section>
			<Button variant="text" icon="logout" onclick={() => (leaveOpen = true)}>Opuść drużynę</Button>
		{/if}

		{#if app.event}
			<section class="stack" aria-labelledby="board-h">
				<div class="row">
					<h2 id="board-h" class="section-title">Ranking</h2>
					<span class="spacer"></span>
					<Chip selected={boardType === 'team'} onclick={() => (boardType = 'team')}>Drużyny</Chip>
					<Chip selected={boardType === 'solo'} onclick={() => (boardType = 'solo')}>Gracze</Chip>
				</div>
				{#if board}
					<ol class="board">
						{#each board.entries.slice(0, 10) as e (e.id)}
							<li class:me={e.isMe}>
								<span class="rank pixel">{e.rank}</span>
								<span class="grow body-medium">{e.name}{e.members ? ` · ${e.members} os.` : ''}</span>
								<span class="pixel label-large">{Math.round(e.score)}</span>
							</li>
						{/each}
					</ol>
				{/if}
			</section>
		{/if}
	</div>

	<Dialog bind:open={inviteOpen} title="Zaproś do drużyny" sheet>
		{#if invite}
			<div class="invite">
				<QrCode text={invite.url} size={240} label="Kod QR zaproszenia do drużyny {team?.name}" />
				<p class="label-large">Kod: <span class="pixel code">{invite.code}</span></p>
				<p class="body-small muted">Ważny jeszcze {Math.floor(left / 60)}:{String(left % 60).padStart(2, '0')} · wolnych miejsc: {invite.usesLeft}</p>
				<div class="row wrap">
					<Button variant="tonal" icon="share" onclick={shareInvite}>Wyślij link</Button>
					<Button variant="text" icon="refresh" onclick={makeInvite}>Nowy kod</Button>
				</div>
			</div>
		{/if}
	</Dialog>

	<Dialog bind:open={leaveOpen} title="Opuścić drużynę?">
		<p>Twoje punkty zostają, ale stracisz bonusy drużynowe za kolejne zadania.</p>
		{#snippet actions()}
			<Button variant="text" onclick={() => (leaveOpen = false)}>Anuluj</Button>
			<Button variant="danger" onclick={leave}>Opuść</Button>
		{/snippet}
	</Dialog>
</AuthGate>

<style>
	.grow {
		flex: 1;
		min-width: 0;
	}
	.error {
		color: var(--md-sys-color-error);
	}
	.emblems {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}
	.em {
		width: 48px;
		height: 48px;
		border-radius: 14px;
		border: 1px solid var(--md-sys-color-outline-variant);
		background: var(--md-sys-color-surface);
		color: var(--md-sys-color-on-surface-variant);
		display: grid;
		place-items: center;
	}
	.em.on {
		background: var(--md-sys-color-primary);
		color: var(--md-sys-color-on-primary);
		border-color: transparent;
	}
	.team-head {
		display: flex;
		align-items: center;
		gap: 16px;
	}
	.emblem {
		width: 56px;
		height: 56px;
		border-radius: 16px;
		display: grid;
		place-items: center;
		background: var(--md-sys-color-primary);
		color: var(--md-sys-color-on-primary);
	}
	.members {
		list-style: none;
		margin: 0;
		padding: 0;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.members li {
		display: flex;
		align-items: center;
		gap: 16px;
		padding: 10px 12px;
		border-radius: var(--md-sys-shape-corner-large);
		background: var(--md-sys-color-surface-container-low);
	}
	.board {
		list-style: none;
		margin: 0;
		padding: 0;
		border-radius: var(--md-sys-shape-corner-large);
		overflow: hidden;
		border: 1px solid var(--md-sys-color-outline-variant);
	}
	.board li {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 12px 16px;
	}
	.board li + li {
		border-top: 1px solid var(--md-sys-color-outline-variant);
	}
	.board li.me {
		background: var(--md-sys-color-primary-container);
		color: var(--md-sys-color-on-primary-container);
	}
	.rank {
		width: 28px;
		color: var(--md-sys-color-primary);
	}
	.invite {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 12px;
		text-align: center;
	}
	.code {
		font-size: 20px;
		letter-spacing: 0.12em;
	}
</style>
