<script lang="ts">
	import type { App, AppEnvVar, AppHealthSummary, Deployment } from '$lib/api';
	import { apiDELETE, apiGET, apiPATCH, apiPOST, prettyDate, shortSHA } from '$lib/api';
	import { onMount } from 'svelte';

	type HistoryResponse = {
		app_id: number;
		app_name: string;
		repo: string;
		branch: string;
		deployments: Deployment[];
	};

	type EnvVarsResponse = {
		app_id: number;
		env_vars: AppEnvVar[];
	};

	type Tab = 'deploys' | 'env' | 'health';
	type EnvDraft = {
		key: string;
		value: string;
		isSecret: boolean;
	};

	export let data: { appID: string };

	let userID = '1';
	let loading = false;
	let error = '';
	let activeTab: Tab = 'deploys';

	let app: App | null = null;
	let history: HistoryResponse | null = null;
	let envVars: AppEnvVar[] = [];
	let health: AppHealthSummary | null = null;

	let envActionPending = false;
	let envActionError = '';
	let envActionSuccess = '';

	let newKey = '';
	let newValue = '';
	let newIsSecret = false;
	let envDrafts: Record<number, EnvDraft> = {};

	$: latest = history?.deployments?.[0] ?? null;

	async function loadHistoryAndApp() {
		app = await apiGET<App>(`/v1/apps/${data.appID}`, userID);
		history = await apiGET<HistoryResponse>(`/v1/apps/${data.appID}/deploys`, userID);
	}

	async function loadEnvVars() {
		const envData = await apiGET<EnvVarsResponse>(`/v1/apps/${data.appID}/env-vars`, userID);
		envVars = envData.env_vars ?? [];
		hydrateEnvDrafts();
	}

	async function loadHealth() {
		health = await apiGET<AppHealthSummary>(`/v1/apps/${data.appID}/health`, userID);
	}

	async function loadPage() {
		loading = true;
		error = '';
		envActionError = '';
		envActionSuccess = '';
		try {
			await Promise.all([loadHistoryAndApp(), loadEnvVars(), loadHealth()]);
		} catch (err) {
			error = err instanceof Error ? err.message : 'failed to load app details';
			app = null;
			history = null;
			envVars = [];
			health = null;
		} finally {
			loading = false;
		}
	}

	function hydrateEnvDrafts() {
		const next: Record<number, EnvDraft> = {};
		for (const envVar of envVars) {
			next[envVar.id] = {
				key: envVar.key,
				value: '',
				isSecret: envVar.is_secret
			};
		}
		envDrafts = next;
	}

	function updateDraft(id: number, patch: Partial<EnvDraft>) {
		const current = envDrafts[id] ?? { key: '', value: '', isSecret: false };
		envDrafts = {
			...envDrafts,
			[id]: {
				...current,
				...patch
			}
		};
	}

	function resetDraft(envVar: AppEnvVar) {
		updateDraft(envVar.id, { key: envVar.key, value: '', isSecret: envVar.is_secret });
	}

	async function createEnvVar() {
		envActionPending = true;
		envActionError = '';
		envActionSuccess = '';
		try {
			await apiPOST<AppEnvVar>(
				`/v1/apps/${data.appID}/env-vars`,
				{
					key: newKey.trim(),
					value: newValue,
					is_secret: newIsSecret
				},
				userID
			);
			newKey = '';
			newValue = '';
			newIsSecret = false;
			await Promise.all([loadEnvVars(), loadHealth()]);
			envActionSuccess = 'Environment variable created.';
		} catch (err) {
			envActionError = err instanceof Error ? err.message : 'failed to create environment variable';
		} finally {
			envActionPending = false;
		}
	}

	async function saveEnvVar(envVar: AppEnvVar) {
		const draft = envDrafts[envVar.id];
		if (!draft) return;

		const payload: Record<string, unknown> = {
			key: draft.key.trim(),
			is_secret: draft.isSecret
		};
		if (draft.value !== '') {
			payload.value = draft.value;
		}

		envActionPending = true;
		envActionError = '';
		envActionSuccess = '';
		try {
			await apiPATCH<AppEnvVar>(`/v1/apps/${data.appID}/env-vars/${envVar.id}`, payload, userID);
			await Promise.all([loadEnvVars(), loadHealth()]);
			envActionSuccess = `Saved ${draft.key.trim()}.`;
		} catch (err) {
			envActionError = err instanceof Error ? err.message : 'failed to update environment variable';
		} finally {
			envActionPending = false;
		}
	}

	async function deleteEnvVar(envVar: AppEnvVar) {
		envActionPending = true;
		envActionError = '';
		envActionSuccess = '';
		try {
			await apiDELETE(`/v1/apps/${data.appID}/env-vars/${envVar.id}`, userID);
			await Promise.all([loadEnvVars(), loadHealth()]);
			envActionSuccess = `Deleted ${envVar.key}.`;
		} catch (err) {
			envActionError = err instanceof Error ? err.message : 'failed to delete environment variable';
		} finally {
			envActionPending = false;
		}
	}

	onMount(loadPage);
</script>

<section class="page">
	<div class="toolbar">
		<div>
			<a href="/apps" class="back">← Back to apps</a>
			<h1>App Control Plane</h1>
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
		<p class="muted">Loading app details...</p>
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
				<p><strong>Current URL:</strong> {health?.current_url || latest?.site_url || app.site_url || 'n/a'}</p>
			</article>
			<article>
				<h2>Latest Deploy</h2>
				<p><strong>Status:</strong> {latest?.status || 'n/a'}</p>
				<p><strong>Trigger:</strong> {latest?.trigger_type || 'n/a'}</p>
				<p>
					<strong>Commit:</strong> {shortSHA(latest?.commit_sha)} {latest?.commit_author ? `by ${latest.commit_author}` : ''}
				</p>
				<p><strong>Updated:</strong> {prettyDate(latest?.updated_at)}</p>
			</article>
		</div>

		<div class="tabs">
			<button class:active={activeTab === 'deploys'} on:click={() => (activeTab = 'deploys')}>Deployments</button>
			<button class:active={activeTab === 'env'} on:click={() => (activeTab = 'env')}>Env Vars</button>
			<button class:active={activeTab === 'health'} on:click={() => (activeTab = 'health')}>Health</button>
		</div>

		{#if activeTab === 'deploys'}
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
		{:else if activeTab === 'env'}
			<div class="env-panel">
				<h2>Environment Variables</h2>
				<p class="muted">Secret values are masked on read. Leave update value blank to keep existing value.</p>

				{#if envActionError}
					<p class="error">{envActionError}</p>
				{/if}
				{#if envActionSuccess}
					<p class="success">{envActionSuccess}</p>
				{/if}

				<div class="create-env">
					<label>
						Key
						<input placeholder="API_TOKEN" bind:value={newKey} />
					</label>
					<label>
						Value
						<input placeholder="value" bind:value={newValue} />
					</label>
					<label class="checkbox">
						<input type="checkbox" bind:checked={newIsSecret} />
						Secret
					</label>
					<button disabled={envActionPending} on:click={createEnvVar}>Add</button>
				</div>

				{#if envVars.length === 0}
					<p class="muted">No environment variables configured.</p>
				{:else}
					<table>
						<thead>
							<tr>
								<th>Key</th>
								<th>Current Value</th>
								<th>Secret</th>
								<th>New Value</th>
								<th>Actions</th>
							</tr>
						</thead>
						<tbody>
							{#each envVars as envVar}
								<tr>
									<td>
										<input
											value={envDrafts[envVar.id]?.key ?? envVar.key}
											on:input={(event) =>
												updateDraft(envVar.id, { key: (event.currentTarget as HTMLInputElement).value })}
										/>
									</td>
									<td>{envVar.value || 'n/a'}</td>
									<td>
										<label class="checkbox">
											<input
												type="checkbox"
												checked={envDrafts[envVar.id]?.isSecret ?? envVar.is_secret}
												on:change={(event) =>
													updateDraft(envVar.id, {
														isSecret: (event.currentTarget as HTMLInputElement).checked
													})}
											/>
										</label>
									</td>
									<td>
										<input
											placeholder={envVar.is_secret ? 'leave blank to keep secret' : 'leave blank to keep value'}
											value={envDrafts[envVar.id]?.value ?? ''}
											on:input={(event) =>
												updateDraft(envVar.id, { value: (event.currentTarget as HTMLInputElement).value })}
										/>
									</td>
									<td>
										<div class="row-actions">
											<button disabled={envActionPending} on:click={() => saveEnvVar(envVar)}>Save</button>
											<button disabled={envActionPending} class="ghost" on:click={() => resetDraft(envVar)}>Reset</button>
											<button disabled={envActionPending} class="danger" on:click={() => deleteEnvVar(envVar)}>Delete</button>
										</div>
									</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}
			</div>
		{:else if activeTab === 'health'}
			{#if !health}
				<p class="muted">No health summary available.</p>
			{:else}
				<div class="health-grid">
					<article>
						<h2>Health Status</h2>
						<p>
							<strong>Indicator:</strong>
							<span class={`indicator ${health.health_indicator}`}>{health.health_indicator}</span>
						</p>
						<p><strong>Current URL:</strong> {health.current_url || 'n/a'}</p>
						<p><strong>Latest Deploy Status:</strong> {health.latest_deploy_status || 'unknown'}</p>
						<p><strong>Alarm State:</strong> {health.alarm_state || 'n/a'}</p>
					</article>
					<article>
						<h2>Reliability Metrics</h2>
						<p><strong>Successes:</strong> {health.metrics.success_count}</p>
						<p><strong>Failures:</strong> {health.metrics.failure_count}</p>
						<p><strong>Total:</strong> {health.metrics.total_count}</p>
						<p><strong>Success Rate:</strong> {health.metrics.success_rate.toFixed(1)}%</p>
					</article>
					<article>
						<h2>Last Successful Deploy</h2>
						{#if health.last_successful_deploy}
							<p><strong>ID:</strong> {health.last_successful_deploy.id}</p>
							<p><strong>Status:</strong> {health.last_successful_deploy.status}</p>
							<p><strong>Updated:</strong> {prettyDate(health.last_successful_deploy.updated_at)}</p>
							<p><strong>Site URL:</strong> {health.last_successful_deploy.site_url || 'n/a'}</p>
						{:else}
							<p class="muted">No successful deployments yet.</p>
						{/if}
					</article>
				</div>
			{/if}
		{/if}
	{/if}
</section>

<style>
	.page {
		padding: 2rem;
		max-width: 1100px;
		margin: 0 auto;
	}
	.back {
		display: inline-block;
		margin-bottom: 0.4rem;
		color: #a9b6e8;
		text-decoration: none;
	}
	.toolbar {
		display: flex;
		justify-content: space-between;
		gap: 1rem;
		align-items: end;
		margin-bottom: 1.2rem;
		flex-wrap: wrap;
	}
	.controls {
		display: flex;
		gap: 0.8rem;
		align-items: end;
	}
	input {
		background: var(--crust);
		border: 1px solid var(--hr-color);
		color: var(--text-color);
		border-radius: 8px;
		padding: 0.5rem;
	}
	.controls input {
		width: 90px;
		margin-left: 0.4rem;
	}
	button {
		background: var(--text-color);
		color: var(--crust);
		border: 0;
		border-radius: 8px;
		padding: 0.55rem 0.8rem;
		cursor: pointer;
	}
	button:disabled {
		opacity: 0.65;
		cursor: not-allowed;
	}
	button.ghost {
		background: transparent;
		color: var(--text-color);
		border: 1px solid #4e5277;
	}
	button.danger {
		background: #7d2430;
		color: #ffd9de;
	}
	.summary-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
		gap: 1rem;
		margin-bottom: 1rem;
	}
	article {
		background: #202236;
		border: 1px solid #2e314f;
		border-radius: 12px;
		padding: 1rem;
	}
	.tabs {
		display: flex;
		gap: 0.5rem;
		margin-bottom: 1rem;
		flex-wrap: wrap;
	}
	.tabs button {
		background: #1f2237;
		color: #bcc5f5;
		border: 1px solid #383d61;
	}
	.tabs button.active {
		background: #b8c3ff;
		color: #121425;
		border-color: #b8c3ff;
	}
	table {
		width: 100%;
		border-collapse: collapse;
		background: #1f2135;
		border-radius: 10px;
		overflow: hidden;
	}
	th,
	td {
		text-align: left;
		padding: 0.6rem;
		border-bottom: 1px solid #2f3357;
		vertical-align: middle;
	}
	th {
		background: #282b45;
	}
	td input {
		width: 100%;
		box-sizing: border-box;
	}
	.row-actions {
		display: flex;
		gap: 0.45rem;
		flex-wrap: wrap;
	}
	.env-panel {
		display: grid;
		gap: 0.8rem;
	}
	.create-env {
		display: grid;
		grid-template-columns: 2fr 3fr auto auto;
		gap: 0.7rem;
		align-items: end;
	}
	.create-env label {
		display: grid;
		gap: 0.3rem;
	}
	.checkbox {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
	}
	.checkbox input {
		width: auto;
	}
	.health-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
		gap: 1rem;
	}
	.indicator {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		padding: 0.1rem 0.55rem;
		border-radius: 999px;
		margin-left: 0.4rem;
		font-size: 0.86rem;
		font-weight: 700;
		text-transform: uppercase;
	}
	.indicator.healthy {
		background: #224d34;
		color: #9ef0b5;
	}
	.indicator.degraded {
		background: #5a4b24;
		color: #f7d58c;
	}
	.indicator.unhealthy {
		background: #5f212c;
		color: #ffb1bf;
	}
	.indicator.unknown {
		background: #313650;
		color: #d0d6ff;
	}
	a {
		color: #b0bfff;
	}
	.error {
		color: #ff9ca8;
	}
	.success {
		color: #9bf0ae;
	}
	.muted {
		opacity: 0.75;
	}
	@media (max-width: 840px) {
		.create-env {
			grid-template-columns: 1fr;
		}
	}
</style>
