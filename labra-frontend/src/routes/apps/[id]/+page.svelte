<script lang="ts">
	import type {
		App,
		AppEnvVar,
		AppHealthSummary,
		Deployment,
		LogQueryHit,
		ObservabilitySummary,
		ReleaseVersion,
		RollbackEvent
	} from '$lib/api';
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

	type ReleasesResponse = {
		app_id: number;
		current_release_id?: number;
		releases: ReleaseVersion[];
	};

	type RollbacksResponse = {
		app_id: number;
		rollbacks: RollbackEvent[];
	};

	type LogQueryResponse = {
		available: boolean;
		provider: string;
		hits: LogQueryHit[];
		reason?: string;
	};

	type Tab = 'deploys' | 'env' | 'health' | 'observability';
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
	let releases: ReleaseVersion[] = [];
	let currentReleaseID = 0;
	let rollbacks: RollbackEvent[] = [];
	let observability: ObservabilitySummary | null = null;

	let envActionPending = false;
	let envActionError = '';
	let envActionSuccess = '';

	let rollbackPending = false;
	let rollbackError = '';
	let rollbackSuccess = '';

	let logQuery = 'build';
	let logSource: 'local' | 'cloudwatch' = 'local';
	let logQueryPending = false;
	let logQueryError = '';
	let logQueryProvider = '';
	let logQueryReason = '';
	let logQueryResults: LogQueryHit[] = [];

	let newKey = '';
	let newValue = '';
	let newIsSecret = false;
	let envDrafts: Record<number, EnvDraft> = {};

	$: latest = history?.deployments?.[0] ?? null;
	$: statusEntries = observability ? Object.entries(observability.status_counts) : [];
	$: triggerEntries = observability ? Object.entries(observability.trigger_counts) : [];
	$: totalStatusCount = statusEntries.reduce((sum, [, value]) => sum + value, 0);
	$: totalTriggerCount = triggerEntries.reduce((sum, [, value]) => sum + value, 0);
	$: releaseByID = buildReleaseMap(releases);

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

	async function loadReleases() {
		const response = await apiGET<ReleasesResponse>(`/v1/apps/${data.appID}/releases`, userID);
		releases = response.releases ?? [];
		currentReleaseID = response.current_release_id ?? 0;
	}

	async function loadRollbacks() {
		const response = await apiGET<RollbacksResponse>(`/v1/apps/${data.appID}/rollbacks`, userID);
		rollbacks = response.rollbacks ?? [];
	}

	async function loadObservability() {
		observability = await apiGET<ObservabilitySummary>(`/v1/apps/${data.appID}/observability`, userID);
	}

	async function loadPage() {
		loading = true;
		error = '';
		envActionError = '';
		envActionSuccess = '';
		rollbackError = '';
		rollbackSuccess = '';
		try {
			await Promise.all([
				loadHistoryAndApp(),
				loadEnvVars(),
				loadHealth(),
				loadReleases(),
				loadRollbacks(),
				loadObservability()
			]);
		} catch (err) {
			error = err instanceof Error ? err.message : 'failed to load app details';
			app = null;
			history = null;
			envVars = [];
			health = null;
			releases = [];
			rollbacks = [];
			observability = null;
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

	function buildReleaseMap(items: ReleaseVersion[]): Record<number, ReleaseVersion> {
		const out: Record<number, ReleaseVersion> = {};
		for (const release of items) {
			out[release.id] = release;
		}
		return out;
	}

	function releaseLabel(releaseID?: number): string {
		if (!releaseID) return 'n/a';
		const release = releaseByID[releaseID];
		if (!release) return `#${releaseID}`;
		return `v${release.version_number}`;
	}

	function toPercent(value: number, total: number): string {
		if (total <= 0) return '0%';
		return `${Math.max(2, Math.round((value / total) * 100))}%`;
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

	async function rollbackToRelease(targetReleaseID: number) {
		const release = releaseByID[targetReleaseID];
		const label = release ? `v${release.version_number}` : `release #${targetReleaseID}`;
		if (!window.confirm(`Rollback ${app?.name ?? 'app'} to ${label}?`)) {
			return;
		}

		rollbackPending = true;
		rollbackError = '';
		rollbackSuccess = '';
		try {
			const response = await apiPOST<{
				deployment: { id: number };
				from_release_id: number;
				target_release_id: number;
			}>(
				`/v1/apps/${data.appID}/rollback`,
				{
					target_release_id: targetReleaseID,
					reason: 'dashboard rollback'
				},
				userID
			);
			rollbackSuccess = `Rollback queued. Deployment #${response.deployment.id} is processing.`;
			await Promise.all([
				loadHistoryAndApp(),
				loadHealth(),
				loadReleases(),
				loadRollbacks(),
				loadObservability()
			]);
		} catch (err) {
			rollbackError = err instanceof Error ? err.message : 'failed to trigger rollback';
		} finally {
			rollbackPending = false;
		}
	}

	async function runLogQuery() {
		logQueryPending = true;
		logQueryError = '';
		logQueryReason = '';
		logQueryProvider = '';
		try {
			const response = await apiGET<LogQueryResponse>(
				`/v1/apps/${data.appID}/observability/log-query?q=${encodeURIComponent(logQuery)}&limit=50&source=${logSource}`,
				userID
			);
			logQueryResults = response.hits ?? [];
			logQueryProvider = response.provider;
			logQueryReason = response.reason ?? '';
			if (response.available === false && response.reason) {
				logQueryError = response.reason;
			}
		} catch (err) {
			logQueryError = err instanceof Error ? err.message : 'failed to run log query';
			logQueryResults = [];
		} finally {
			logQueryPending = false;
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
				<p><strong>Current Release:</strong> {releaseLabel(currentReleaseID)}</p>
			</article>
			<article>
				<h2>Latest Deploy</h2>
				<p><strong>Status:</strong> {latest?.status || 'n/a'}</p>
				<p><strong>Trigger:</strong> {latest?.trigger_type || 'n/a'}</p>
				<p><strong>Release:</strong> {releaseLabel(latest?.release_id)}</p>
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
			<button class:active={activeTab === 'observability'} on:click={() => (activeTab = 'observability')}>Observability</button>
		</div>

		{#if activeTab === 'deploys'}
			{#if rollbackError}
				<p class="error">{rollbackError}</p>
			{/if}
			{#if rollbackSuccess}
				<p class="success">{rollbackSuccess}</p>
			{/if}

			{#if history.deployments.length === 0}
				<p class="muted">No deployments yet.</p>
			{:else}
				<table>
					<thead>
						<tr>
							<th>ID</th>
							<th>Status</th>
							<th>Trigger</th>
							<th>Release</th>
							<th>Commit</th>
							<th>Updated</th>
							<th>Actions</th>
						</tr>
					</thead>
					<tbody>
						{#each history.deployments as dep}
							<tr>
								<td>{dep.id}</td>
								<td>{dep.status}</td>
								<td>{dep.trigger_type}</td>
								<td>{releaseLabel(dep.release_id)}</td>
								<td title={dep.commit_message || ''}>{shortSHA(dep.commit_sha)}</td>
								<td>{prettyDate(dep.updated_at)}</td>
								<td>
									<div class="row-actions">
										<a href={`/deploys/${dep.id}`}>open</a>
										{#if dep.release_id && dep.release_id !== currentReleaseID}
											<button
												disabled={rollbackPending}
												class="rollback"
												on:click={() => rollbackToRelease(dep.release_id || 0)}
											>
												Rollback
											</button>
										{/if}
									</div>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			{/if}

			<h3>Rollback Timeline</h3>
			{#if rollbacks.length === 0}
				<p class="muted">No rollback events yet.</p>
			{:else}
				<table>
					<thead>
						<tr>
							<th>When</th>
							<th>From</th>
							<th>To</th>
							<th>Deployment</th>
							<th>Reason</th>
						</tr>
					</thead>
					<tbody>
						{#each rollbacks as event}
							<tr>
								<td>{prettyDate(event.created_at)}</td>
								<td>{releaseLabel(event.from_release_id)}</td>
								<td>{releaseLabel(event.to_release_id)}</td>
								<td><a href={`/deploys/${event.deployment_id}`}>#{event.deployment_id}</a></td>
								<td>{event.reason || 'n/a'}</td>
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
						<p><strong>Current Release:</strong> {releaseLabel(health.current_release_id)}</p>
						<p><strong>Latest Deploy Status:</strong> {health.latest_deploy_status || 'unknown'}</p>
						<p><strong>Alarm State:</strong> {health.alarm_state || 'n/a'}</p>
					</article>
					<article>
						<h2>Reliability Metrics</h2>
						<p><strong>Successes:</strong> {health.metrics.success_count}</p>
						<p><strong>Failures:</strong> {health.metrics.failure_count}</p>
						<p><strong>Total:</strong> {health.metrics.total_count}</p>
						<p><strong>Rollback Count:</strong> {health.metrics.rollback_count}</p>
						<p><strong>Success Rate:</strong> {health.metrics.success_rate.toFixed(1)}%</p>
						<p><strong>Avg Duration:</strong> {health.metrics.average_duration_seconds.toFixed(1)}s</p>
						<p><strong>Latest Duration:</strong> {health.metrics.latest_duration_seconds}s</p>
					</article>
					<article>
						<h2>Last Successful Deploy</h2>
						{#if health.last_successful_deploy}
							<p><strong>ID:</strong> {health.last_successful_deploy.id}</p>
							<p><strong>Status:</strong> {health.last_successful_deploy.status}</p>
							<p><strong>Release:</strong> {releaseLabel(health.last_successful_deploy.release_id)}</p>
							<p><strong>Updated:</strong> {prettyDate(health.last_successful_deploy.updated_at)}</p>
							<p><strong>Site URL:</strong> {health.last_successful_deploy.site_url || 'n/a'}</p>
						{:else}
							<p class="muted">No successful deployments yet.</p>
						{/if}
					</article>
				</div>
			{/if}
		{:else if activeTab === 'observability'}
			{#if !observability}
				<p class="muted">No observability data available.</p>
			{:else}
				<div class="observability-grid">
					<article>
						<h2>Status Distribution</h2>
						{#if statusEntries.length === 0}
							<p class="muted">No status data yet.</p>
						{:else}
							{#each statusEntries as [label, value]}
								<div class="metric-row">
									<span>{label}</span>
									<span>{value}</span>
								</div>
								<div class="bar-track">
									<div class="bar-fill" style={`width: ${toPercent(value, totalStatusCount)};`}></div>
								</div>
							{/each}
						{/if}
					</article>
					<article>
						<h2>Trigger Distribution</h2>
						{#if triggerEntries.length === 0}
							<p class="muted">No trigger data yet.</p>
						{:else}
							{#each triggerEntries as [label, value]}
								<div class="metric-row">
									<span>{label}</span>
									<span>{value}</span>
								</div>
								<div class="bar-track">
									<div class="bar-fill trigger" style={`width: ${toPercent(value, totalTriggerCount)};`}></div>
								</div>
							{/each}
						{/if}
					</article>
					<article>
						<h2>Pipeline Snapshot</h2>
						<p><strong>Release Count:</strong> {observability.release_count}</p>
						<p><strong>Current Release:</strong> {releaseLabel(observability.current_release_id)}</p>
						<p><strong>CloudWatch Mode:</strong> {observability.cloudwatch_enabled ? 'enabled' : 'disabled'}</p>
						<p><strong>Health Indicator:</strong> {observability.health_indicator}</p>
						<p><strong>Alarm State:</strong> {observability.alarm_state || 'n/a'}</p>
					</article>
				</div>

				<h3>Recent Durations</h3>
				{#if observability.recent_durations.length === 0}
					<p class="muted">No duration data available.</p>
				{:else}
					<table>
						<thead>
							<tr>
								<th>Deployment</th>
								<th>Status</th>
								<th>Trigger</th>
								<th>Duration</th>
								<th>Finished</th>
							</tr>
						</thead>
						<tbody>
							{#each observability.recent_durations as point}
								<tr>
									<td><a href={`/deploys/${point.deployment_id}`}>#{point.deployment_id}</a></td>
									<td>{point.status}</td>
									<td>{point.trigger_type}</td>
									<td>{point.duration_seconds}s</td>
									<td>{prettyDate(point.finished_at)}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{/if}

				<h3>Log Query</h3>
				<div class="log-query-controls">
					<label>
						Query
						<input bind:value={logQuery} />
					</label>
					<label>
						Source
						<select bind:value={logSource}>
							<option value="local">Local</option>
							<option value="cloudwatch">CloudWatch</option>
						</select>
					</label>
					<button disabled={logQueryPending} on:click={runLogQuery}>Run Query</button>
				</div>
				{#if logQueryProvider}
					<p class="muted"><strong>Provider:</strong> {logQueryProvider}</p>
				{/if}
				{#if logQueryReason}
					<p class="muted">{logQueryReason}</p>
				{/if}
				{#if logQueryError}
					<p class="error">{logQueryError}</p>
				{/if}
				{#if logQueryResults.length > 0}
					<table>
						<thead>
							<tr>
								<th>When</th>
								<th>Deployment</th>
								<th>Level</th>
								<th>Message</th>
							</tr>
						</thead>
						<tbody>
							{#each logQueryResults as hit}
								<tr>
									<td>{prettyDate(hit.created_at)}</td>
									<td><a href={`/deploys/${hit.deployment_id}`}>#{hit.deployment_id}</a></td>
									<td>{hit.log_level}</td>
									<td>{hit.message}</td>
								</tr>
							{/each}
						</tbody>
					</table>
				{:else if logQueryProvider}
					<p class="muted">No log hits found for this query.</p>
				{/if}
			{/if}
		{/if}
	{/if}
</section>

<style>
	.page {
		padding: 2rem;
		max-width: 1200px;
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
	input,
	select {
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
	button.rollback {
		background: #2a5876;
		color: #d9efff;
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
		margin-bottom: 1rem;
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
		align-items: center;
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
	.observability-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
		gap: 1rem;
		margin-bottom: 1rem;
	}
	.metric-row {
		display: flex;
		justify-content: space-between;
		margin-top: 0.5rem;
		font-size: 0.95rem;
	}
	.bar-track {
		height: 8px;
		border-radius: 999px;
		background: #2f3456;
		margin-top: 0.2rem;
		overflow: hidden;
	}
	.bar-fill {
		height: 100%;
		background: #6f86ff;
		border-radius: 999px;
	}
	.bar-fill.trigger {
		background: #4ec7ad;
	}
	.log-query-controls {
		display: grid;
		grid-template-columns: 2fr 1fr auto;
		gap: 0.7rem;
		align-items: end;
		margin-bottom: 0.8rem;
	}
	.log-query-controls label {
		display: grid;
		gap: 0.3rem;
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
	h3 {
		margin: 0.6rem 0;
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
	@media (max-width: 900px) {
		.create-env,
		.log-query-controls {
			grid-template-columns: 1fr;
		}
	}
</style>
