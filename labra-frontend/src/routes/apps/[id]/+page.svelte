<script lang="ts">
	import type { App, Deployment } from '$lib/api';
	import { apiGET, prettyDate, shortSHA } from '$lib/api';
	import { onMount } from 'svelte';

	type HistoryResponse = {
		app_id: number;
		app_name: string;
		repo: string;
		branch: string;
		deployments: Deployment[];
	};

	export let data: { appID: string };

	let userID = '1';
	let loading = false;
	let error = '';
	let app: App | null = null;
	let history: HistoryResponse | null = null;

	$: latest = history?.deployments?.[0] ?? null;

	async function loadPage() {
		loading = true;
		error = '';
		try {
			app = await apiGET<App>(`/v1/apps/${data.appID}`, userID);
			history = await apiGET<HistoryResponse>(`/v1/apps/${data.appID}/deploys`, userID);
		} catch (err) {
			error = err instanceof Error ? err.message : 'failed to load app details';
			app = null;
			history = null;
		} finally {
			loading = false;
		}
	}

	onMount(loadPage);
</script>

<section class="page">
	<div class="toolbar">
		<div>
			<a href="/apps" class="back">← Back to apps</a>
			<h1>App Deploy History</h1>
		</div>
		<div class="controls">
			<label>
				User ID
				<input bind:value={userID} />
			</label>
			<button on:click={loadPage}>Refresh</button>
		</div>
	</div>

	{#if loading}
		<p class="muted">Loading app history...</p>
	{:else if error}
		<p class="error">{error}</p>
	{:else if !app || !history}
		<p class="muted">No app data.</p>
	{:else}
		<div class="summary-grid">
			<article>
				<h2>{app.name}</h2>
				<p><strong>Repo:</strong> {app.repo_full_name}</p>
				<p><strong>Branch:</strong> {app.branch}</p>
				<p><strong>Current Site URL:</strong> {latest?.site_url || app.site_url || 'n/a'}</p>
			</article>
			<article>
				<h2>Latest Deploy</h2>
				<p><strong>Status:</strong> {latest?.status || 'n/a'}</p>
				<p><strong>Trigger:</strong> {latest?.trigger_type || 'n/a'}</p>
				<p><strong>Commit:</strong> {shortSHA(latest?.commit_sha)} {latest?.commit_author ? `by ${latest.commit_author}` : ''}</p>
				<p><strong>Updated:</strong> {prettyDate(latest?.updated_at)}</p>
			</article>
		</div>

		{#if history.deployments.length === 0}
			<p class="muted">No deployments yet.</p>
		{:else}
			<table>
				<thead>
					<tr>
						<th>ID</th>
						<th>Status</th>
						<th>Trigger</th>
						<th>Commit</th>
						<th>Updated</th>
						<th>Details</th>
					</tr>
				</thead>
				<tbody>
					{#each history.deployments as dep}
						<tr>
							<td>{dep.id}</td>
							<td>{dep.status}</td>
							<td>{dep.trigger_type}</td>
							<td title={dep.commit_message || ''}>{shortSHA(dep.commit_sha)}</td>
							<td>{prettyDate(dep.updated_at)}</td>
							<td><a href={`/deploys/${dep.id}`}>open</a></td>
						</tr>
					{/each}
				</tbody>
			</table>
		{/if}
	{/if}
</section>

<style>
	.page { padding: 2rem; max-width: 1100px; margin: 0 auto; }
	.back { display: inline-block; margin-bottom: 0.4rem; color: #a9b6e8; text-decoration: none; }
	.toolbar { display: flex; justify-content: space-between; gap: 1rem; align-items: end; margin-bottom: 1.2rem; flex-wrap: wrap; }
	.controls { display: flex; gap: 0.8rem; align-items: end; }
	input { background: var(--crust); border: 1px solid var(--hr-color); color: var(--text-color); border-radius: 8px; padding: 0.5rem; width: 90px; margin-left: 0.4rem; }
	button { background: var(--text-color); color: var(--crust); border: 0; border-radius: 8px; padding: 0.55rem 0.8rem; cursor: pointer; }
	.summary-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 1rem; margin-bottom: 1rem; }
	article { background: #202236; border: 1px solid #2e314f; border-radius: 12px; padding: 1rem; }
	table { width: 100%; border-collapse: collapse; background: #1f2135; border-radius: 10px; overflow: hidden; }
	th, td { text-align: left; padding: 0.6rem; border-bottom: 1px solid #2f3357; }
	th { background: #282b45; }
	a { color: #b0bfff; }
	.error { color: #ff9ca8; }
	.muted { opacity: 0.75; }
</style>
