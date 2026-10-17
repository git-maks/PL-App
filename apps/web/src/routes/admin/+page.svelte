<script lang="ts">
	import { app, errorMessage } from '#lib/state/app.svelte';
	import { href } from '#lib/paths';
	import { toast } from '#lib/state/toast.svelte';
	import { statByCode } from '#lib/content/archetypes';
	import { STAT_CODES, type StatCode } from '#lib/game/types';
	import type { Analytics, Station } from '#lib/api';
	import AuthGate from '#lib/components/AuthGate.svelte';
	import TopAppBar from '#lib/components/TopAppBar.svelte';
	import Card from '#lib/components/Card.svelte';
	import Button from '#lib/components/Button.svelte';
	import TextField from '#lib/components/TextField.svelte';
	import Skeleton from '#lib/components/Skeleton.svelte';
	import StatIcon from '#lib/components/StatIcon.svelte';

	let data = $state<Analytics | null>(null);
	let error = $state<string | null>(null);
	let title = $state('');
	let body = $state('');
	let roleEmail = $state('');
	let role = $state<'participant' | 'staff' | 'admin'>('staff');
	let editId = $state('');
	let weights = $state<Record<StatCode, number>>({ STR: 0, INT: 0, MAN: 0, KRE: 0, ENE: 0, CHA: 0, WIS: 0, VIT: 0 });
	let basePoints = $state(100);
	let staffEmails = $state('');

	$effect(() => {
		if (app.me && app.event && !data) {
			app.api.analytics(app.event.id).then((d) => (data = d)).catch((e) => (error = errorMessage(e)));
		}
	});
	const maxVisits = $derived(Math.max(1, ...(data?.stations.map((s) => s.visits) ?? [1])));
	const maxArch = $derived(Math.max(1, ...(data?.archetypes.map((a) => a.users) ?? [1])));
	const csv = $derived(app.event ? app.api.analyticsCsvUrl(app.event.id) : null);
	const editing = $derived<Station | undefined>(app.stationById.get(editId));
	const sum = $derived(STAT_CODES.reduce((a, c) => a + (weights[c] || 0), 0));

	function pick(id: string) {
		editId = id;
		const st = app.stationById.get(id);
		if (!st) return;
		basePoints = st.basePoints;
		weights = Object.fromEntries(STAT_CODES.map((c) => [c, Math.round((st.weights[c] ?? 0) * 100)])) as Record<StatCode, number>;
	}
	async function saveWeights() {
		if (!editing || sum <= 0) return;
		const normalized = Object.fromEntries(STAT_CODES.filter((c) => weights[c] > 0).map((c) => [c, Math.round((weights[c] / sum) * 1000) / 1000]));
		try {
			const emails = staffEmails.split(/[\s,;]+/).map((x) => x.trim()).filter((x) => x.includes('@'));
			await app.api.patchStation(editing.id, { basePoints, weights: normalized, ...(emails.length ? { staffEmails: emails } : {}) });
			app.stations = await app.api.stations(app.event!.id);
			toast('Zapisano wagi stacji.');
		} catch (e) {
			toast(errorMessage(e));
		}
	}
	async function sendBroadcast(e: SubmitEvent) {
		e.preventDefault();
		try {
			await app.api.broadcast(title, body);
			title = body = '';
			toast('Komunikat wysłany.');
		} catch (err) {
			toast(errorMessage(err));
		}
	}
	async function grant(e: SubmitEvent) {
		e.preventDefault();
		try {
			await app.api.setRole(roleEmail, role);
			toast('Uprawnienia zmienione.');
		} catch (err) {
			toast(errorMessage(err));
		}
	}
</script>

<svelte:head><title>Administracja · UniQuest</title></svelte:head>

<AuthGate>
	<TopAppBar title="Administracja" back={href('/settings')} />
	<div class="page">
		{#if app.me?.role !== 'admin'}
			<p>Brak uprawnień administratora.</p>
		{:else if !app.event}
			<p>Brak aktywnego wydarzenia.</p>
		{:else}
			<h2 class="section-title">{app.event.name}</h2>
			{#if error}<p role="alert">{error}</p>{:else if !data}<Skeleton lines={4} />{:else}
				<div class="kpis">
					<div class="kpi"><span class="pixel big">{data.participants}</span><span class="label-medium">uczestników</span></div>
					<div class="kpi"><span class="pixel big">{data.scans}</span><span class="label-medium">skanów</span></div>
					<div class="kpi"><span class="pixel big">{data.scans ? Math.round((data.rated / data.scans) * 100) : 0}%</span><span class="label-medium">ocenionych</span></div>
				</div>

				<h3 class="title-small">Zainteresowanie stacjami</h3>
				<Card variant="outlined">
					<table class="tbl body-small">
						<thead><tr><th>Stacja</th><th>Wizyty</th><th>Śr. ocena</th></tr></thead>
						<tbody>
							{#each [...data.stations].sort((a, b) => (b.avgRating ?? 0) - (a.avgRating ?? 0)) as s (s.stationId)}
								<tr>
									<td>{s.name}<span class="bar" style="width: {(s.visits / maxVisits) * 100}%"></span></td>
									<td>{s.visits}</td>
									<td>{s.avgRating ? s.avgRating.toFixed(2) : '—'}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				</Card>

				<h3 class="title-small">Dominujące archetypy</h3>
				<Card variant="outlined">
					<div class="arch">
						{#each data.archetypes as a (a.code)}
							<div class="arow" style="--c: {statByCode[a.code].color}">
								<StatIcon code={a.code} size={18} />
								<span class="body-small name">{statByCode[a.code].archetype}</span>
								<span class="abar"><span style="width: {(a.users / maxArch) * 100}%"></span></span>
								<span class="label-medium">{a.users}</span>
							</div>
						{/each}
					</div>
				</Card>
				{#if csv}<Button variant="tonal" icon="download" href={csv}>Eksport CSV (anonimowy)</Button>{/if}
			{/if}

			<h2 class="section-title">Wagi stacji</h2>
			<Card variant="filled">
				<div class="stack">
					<label class="body-medium sel">
						Stacja
						<select bind:value={editId} onchange={() => pick(editId)}>
							<option value="">— wybierz —</option>
							{#each app.stations as st (st.id)}<option value={st.id}>{st.name}</option>{/each}
						</select>
					</label>
					{#if editing}
						<label class="body-medium sel">Punkty bazowe <input type="number" min="1" max="1000" bind:value={basePoints} /></label>
						<div class="weights">
							{#each STAT_CODES as c (c)}
								<label class="w" style="--c: {statByCode[c].color}">
									<StatIcon code={c} size={16} /><span class="label-medium">{statByCode[c].name}</span>
									<input type="range" min="0" max="100" step="5" bind:value={weights[c]} />
									<span class="label-medium num">{sum ? Math.round((weights[c] / sum) * 100) : 0}%</span>
								</label>
							{/each}
						</div>
						<TextField label="E-maile obsługi stacji (po przecinku)" bind:value={staffEmails} supporting="Zweryfikowane konta dostaną dostęp do panelu tej stacji" />
						<Button icon="check" disabled={sum === 0} onclick={saveWeights}>Zapisz (wagi normalizujemy do 100%)</Button>
					{/if}
				</div>
			</Card>

			<h2 class="section-title">Komunikat do uczestników</h2>
			<Card variant="filled">
				<form class="stack" onsubmit={sendBroadcast}>
					<TextField label="Tytuł" bind:value={title} maxlength={80} />
					<TextField label="Treść" bind:value={body} maxlength={280} />
					<Button type="submit" icon="campaign" disabled={!title || !body}>Wyślij (w aplikacji i push)</Button>
				</form>
			</Card>

			<h2 class="section-title">Uprawnienia</h2>
			<Card variant="filled">
				<form class="stack" onsubmit={grant}>
					<TextField type="email" label="E-mail (zweryfikowane konto)" bind:value={roleEmail} />
					<select bind:value={role} aria-label="Rola">
						<option value="staff">Obsługa stacji</option>
						<option value="admin">Administrator</option>
						<option value="participant">Uczestnik (odbierz uprawnienia)</option>
					</select>
					<Button type="submit" icon="admin_panel_settings" disabled={!roleEmail.includes('@')}>Zmień rolę</Button>
				</form>
			</Card>
		{/if}
	</div>
</AuthGate>

<style>
	.kpis {
		display: grid;
		grid-template-columns: repeat(3, 1fr);
		gap: 8px;
	}
	.kpi {
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 16px 8px;
		border-radius: var(--md-sys-shape-corner-large);
		background: var(--md-sys-color-primary-container);
		color: var(--md-sys-color-on-primary-container);
	}
	.big {
		font-size: 1.75rem;
	}
	.tbl {
		width: 100%;
		border-collapse: collapse;
	}
	.tbl th {
		text-align: left;
		font: var(--md-sys-typescale-label-medium);
		color: var(--md-sys-color-on-surface-variant);
		padding: 4px;
	}
	.tbl td {
		padding: 6px 4px;
		border-top: 1px solid var(--md-sys-color-outline-variant);
		vertical-align: top;
	}
	.tbl td:not(:first-child) {
		text-align: right;
		white-space: nowrap;
	}
	.bar {
		display: block;
		height: 4px;
		margin-top: 4px;
		border-radius: 2px;
		background: var(--md-sys-color-primary);
	}
	.arch {
		display: flex;
		flex-direction: column;
		gap: 8px;
	}
	.arow {
		display: grid;
		grid-template-columns: 18px 110px 1fr 36px;
		align-items: center;
		gap: 8px;
		color: var(--c);
	}
	.arow .name,
	.arow .label-medium {
		color: var(--md-sys-color-on-surface);
	}
	.abar {
		height: 10px;
		border-radius: 5px;
		background: var(--md-sys-color-surface-container-highest);
		overflow: hidden;
	}
	.abar span {
		display: block;
		height: 100%;
		background: var(--c);
	}
	select,
	input[type='number'] {
		height: 48px;
		padding: 0 12px;
		border-radius: var(--md-sys-shape-corner-extra-small);
		border: 1px solid var(--md-sys-color-outline);
		background: var(--md-sys-color-surface);
	}
	.sel {
		display: flex;
		flex-direction: column;
		gap: 4px;
	}
	.weights {
		display: flex;
		flex-direction: column;
		gap: 6px;
	}
	.w {
		display: grid;
		grid-template-columns: 16px 90px 1fr 40px;
		align-items: center;
		gap: 8px;
		color: var(--c);
	}
	.w input {
		accent-color: var(--c);
	}
	.num {
		text-align: right;
		color: var(--md-sys-color-on-surface);
	}
</style>
