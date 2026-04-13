<script lang="ts">
	import type { Deployment, DeploymentLog } from '$lib/api';
	import { apiGET, prettyDate, shortSHA } from '$lib/api';
	import { onMount } from 'svelte';

	export let data: { deployID: string };

	let userID = '1';
	let loading = false;
	let error = '';
	let deploy: Deployment | null = null;
	let logs: DeploymentLog[] = [];

	async function loadPage() {
		loading = true;
		error = '';
		try {
			deploy = await apiGET<Deployment>(`/v1/deploys/${data.deployID}`, userID);
			const logRes = await apiGET<{ logs: DeploymentLog[] }>(`/v1/deploys/${data.deployID}/logs`, userID);
			logs = logRes.logs ?? [];
		} catch (err) {
			error = err instanceof Error ? err.message : 'failed to load deploy details';
			deploy = null;
			logs = [];
		} finally {
			loading = false;
		}
	}

	onMount(loadPage);
</script>

<section class="page">
	<div class="toolbar">
		<div>
			<h1>Deployment #{data.deployID}</h1>
			{#if deploy}
				<a class="back" href={`/apps/${deploy.app_id}`}>← Back to app history</a>
			{/if}
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
		<p class="muted">Loading deployment...</p>
	{:else if error}
		<p class="error">{error}</p>
	{:else if !deploy}
		<p class="muted">No deployment found.</p>
	{:else}
		<div class="summary-grid">
			<article>
				<h2>Status</h2>
				<p><strong>{deploy.status}</strong></p>
				<p><strong>Trigger:</strong> {deploy.trigger_type}</p>
				<p><strong>Updated:</strong> {prettyDate(deploy.updated_at)}</p>
				<p><strong>Site URL:</strong> {deploy.site_url || 'n/a'}</p>
			</article>
			<article>
				<h2>Commit</h2>
				<p><strong>SHA:</strong> {shortSHA(deploy.commit_sha)}</p>
				<p><strong>Author:</strong> {deploy.commit_author || 'n/a'}</p>
				<p><strong>Message:</strong> {deploy.commit_message || 'n/a'}</p>
				<p><strong>Branch:</strong> {deploy.branch || 'n/a'}</p>
			</article>
		</div>

		<h2>Logs</h2>
		{#if logs.length === 0}
			<p class="muted">No logs for this deployment yet.</p>
		{:else}
			<ul class="logs">
				{#each logs as log}
					<li>
						<span class="stamp">[{prettyDate(log.created_at)}]</span>
						<span class="level">{log.log_level.toUpperCase()}</span>
						<span>{log.message}</span>
					</li>
				{/each}
			</ul>
		{/if}
	{/if}
</section>

<style>
	.page { padding: 2rem; max-width: 1100px; margin: 0 auto; }
	.toolbar { display: flex; justify-content: space-between; gap: 1rem; align-items: end; margin-bottom: 1.2rem; flex-wrap: wrap; }
	.controls { display: flex; gap: 0.8rem; align-items: end; }
	.back { display: inline-block; margin-top: 0.2rem; color: #a9b6e8; text-decoration: none; }
	input { background: var(--crust); border: 1px solid var(--hr-color); color: var(--text-color); border-radius: 8px; padding: 0.5rem; width: 90px; margin-left: 0.4rem; }
	button { background: var(--text-color); color: var(--crust); border: 0; border-radius: 8px; padding: 0.55rem 0.8rem; cursor: pointer; }
	.summary-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(260px, 1fr)); gap: 1rem; margin-bottom: 1rem; }
	article { background: #202236; border: 1px solid #2e314f; border-radius: 12px; padding: 1rem; }
	.logs { list-style: none; padding: 0; margin: 0; background: #1f2135; border: 1px solid #2f3357; border-radius: 12px; }
	.logs li { padding: 0.6rem 0.8rem; border-bottom: 1px solid #2f3357; display: flex; gap: 0.6rem; flex-wrap: wrap; }
	.logs li:last-child { border-bottom: 0; }
	.stamp { opacity: 0.7; }
	.level { color: #b9c2ff; min-width: 55px; }
	.error { color: #ff9ca8; }
	.muted { opacity: 0.75; }
</style>
