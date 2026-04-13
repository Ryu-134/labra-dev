<script lang="ts">
	import type { App } from '$lib/api';
	import { apiGET, prettyDate } from '$lib/api';
	import { onMount } from 'svelte';

	let userID = '1';
	let loading = false;
	let error = '';
	let apps: App[] = [];

	async function loadApps() {
		loading = true;
		error = '';
		try {
			const data = await apiGET<{ apps: App[] }>('/v1/apps', userID);
			apps = data.apps ?? [];
		} catch (err) {
			error = err instanceof Error ? err.message : 'failed to load apps';
			apps = [];
		} finally {
			loading = false;
		}
	}

	onMount(loadApps);
</script>

<section class="page">
	<div class="toolbar">
		<h1>Apps</h1>
		<div class="controls">
			<label>
				User ID
				<input bind:value={userID} />
			</label>
			<button on:click={loadApps}>Refresh</button>
		</div>
	</div>

	{#if loading}
		<p class="muted">Loading apps...</p>
	{:else if error}
		<p class="error">{error}</p>
	{:else if apps.length === 0}
		<p class="muted">No apps yet.</p>
	{:else}
		<div class="cards">
			{#each apps as app}
				<a class="card" href={`/apps/${app.id}`}>
					<h2>{app.name}</h2>
					<p><strong>Repo:</strong> {app.repo_full_name}</p>
					<p><strong>Branch:</strong> {app.branch}</p>
					<p><strong>Build:</strong> {app.build_type}</p>
					<p><strong>Updated:</strong> {prettyDate(app.updated_at)}</p>
				</a>
			{/each}
		</div>
	{/if}
</section>

<style>
	.page { padding: 2rem; max-width: 1100px; margin: 0 auto; }
	.toolbar { display: flex; justify-content: space-between; gap: 1rem; align-items: end; margin-bottom: 1.2rem; flex-wrap: wrap; }
	.controls { display: flex; gap: 0.8rem; align-items: end; }
	input { background: var(--crust); border: 1px solid var(--hr-color); color: var(--text-color); border-radius: 8px; padding: 0.5rem; width: 90px; margin-left: 0.4rem; }
	button { background: var(--text-color); color: var(--crust); border: 0; border-radius: 8px; padding: 0.55rem 0.8rem; cursor: pointer; }
	.cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 1rem; }
	.card { background: #202236; border: 1px solid #2e314f; border-radius: 12px; padding: 1rem; text-decoration: none; color: var(--text-color); }
	.card:hover { border-color: #5f6ea8; }
	.error { color: #ff9ca8; }
	.muted { opacity: 0.75; }
</style>
