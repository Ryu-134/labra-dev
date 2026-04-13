export type App = {
	id: number;
	user_id: number;
	name: string;
	repo_full_name: string;
	branch: string;
	build_type: string;
	output_dir: string;
	root_dir?: string;
	site_url?: string;
	auto_deploy_enabled: boolean;
	current_release_id?: number;
	created_at: number;
	updated_at: number;
};

export type Deployment = {
	id: number;
	app_id: number;
	user_id: number;
	status: string;
	trigger_type: string;
	commit_sha?: string;
	commit_message?: string;
	commit_author?: string;
	branch?: string;
	site_url?: string;
	failure_reason?: string;
	release_id?: number;
	created_at: number;
	updated_at: number;
	started_at?: number;
	finished_at?: number;
};

export type DeploymentLog = {
	id: number;
	deployment_id: number;
	log_level: string;
	message: string;
	created_at: number;
};

export type AppEnvVar = {
	id: number;
	app_id: number;
	key: string;
	value: string;
	is_secret: boolean;
	masked: boolean;
	created_at: number;
	updated_at: number;
};

export type HealthDeploymentSummary = {
	id: number;
	status: string;
	trigger_type: string;
	site_url?: string;
	commit_sha?: string;
	release_id?: number;
	updated_at: number;
};

export type AppHealthSummary = {
	app_id: number;
	app_name: string;
	repo_full_name: string;
	branch: string;
	current_url: string;
	current_release_id?: number;
	latest_deploy_status: string;
	latest_deploy?: HealthDeploymentSummary;
	last_successful_deploy?: HealthDeploymentSummary;
	metrics: {
		success_count: number;
		failure_count: number;
		total_count: number;
		success_rate: number;
		average_duration_seconds: number;
		latest_duration_seconds: number;
		total_duration_seconds: number;
		rollback_count: number;
		last_deploy_at: number;
		last_success_at: number;
		last_failure_at: number;
	};
	alarm_state?: string;
	health_indicator: string;
};

export type ReleaseVersion = {
	id: number;
	app_id: number;
	deployment_id: number;
	version_number: number;
	artifact_path: string;
	artifact_checksum?: string;
	is_retained: boolean;
	created_at: number;
};

export type RollbackEvent = {
	id: number;
	app_id: number;
	user_id: number;
	from_release_id?: number;
	to_release_id: number;
	deployment_id: number;
	reason?: string;
	created_at: number;
};

export type LogQueryHit = {
	log_id: number;
	deployment_id: number;
	log_level: string;
	message: string;
	created_at: number;
	status: string;
	trigger_type: string;
	release_id?: number;
};

export type ObservabilitySummary = {
	app_id: number;
	current_release_id?: number;
	release_count: number;
	status_counts: Record<string, number>;
	trigger_counts: Record<string, number>;
	recent_durations: Array<{
		deployment_id: number;
		status: string;
		trigger_type: string;
		duration_seconds: number;
		finished_at: number;
	}>;
	recent_failures: Deployment[];
	recent_rollbacks: RollbackEvent[];
	health: {
		success_count: number;
		failure_count: number;
		total_count: number;
		success_rate: number;
		average_duration_seconds: number;
		latest_duration_seconds: number;
		total_duration_seconds: number;
		rollback_count: number;
		last_deploy_at: number;
		last_success_at: number;
		last_failure_at: number;
	};
	health_indicator: string;
	alarm_state?: string;
	cloudwatch_enabled: boolean;
};

export const backendBaseURL = import.meta.env.VITE_BACKEND_BASE_URL ?? '/api';

async function apiRequest<T>(method: string, path: string, userID: string, body?: unknown): Promise<T> {
	const res = await fetch(`${backendBaseURL}${path}`, {
		method,
		headers: {
			'Content-Type': 'application/json',
			'X-User-ID': userID
		},
		body: body === undefined ? undefined : JSON.stringify(body)
	});

	if (!res.ok) {
		let detail = `request failed (${res.status})`;
		try {
			const parsed = await res.json();
			detail = parsed?.error?.message ?? detail;
		} catch {
			// keep fallback error
		}
		throw new Error(detail);
	}

	if (res.status === 204) {
		return undefined as T;
	}

	return (await res.json()) as T;
}

export async function apiGET<T>(path: string, userID: string): Promise<T> {
	return apiRequest<T>('GET', path, userID);
}

export async function apiPOST<T>(path: string, body: unknown, userID: string): Promise<T> {
	return apiRequest<T>('POST', path, userID, body);
}

export async function apiPATCH<T>(path: string, body: unknown, userID: string): Promise<T> {
	return apiRequest<T>('PATCH', path, userID, body);
}

export async function apiDELETE(path: string, userID: string): Promise<void> {
	return apiRequest<void>('DELETE', path, userID);
}

export function shortSHA(sha?: string): string {
	if (!sha) return 'n/a';
	return sha.slice(0, 7);
}

export function prettyDate(epochSeconds?: number): string {
	if (!epochSeconds || epochSeconds <= 0) return 'n/a';
	return new Date(epochSeconds * 1000).toLocaleString();
}
