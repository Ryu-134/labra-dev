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

export const backendBaseURL = import.meta.env.VITE_BACKEND_BASE_URL ?? '/api';

export async function apiGET<T>(path: string, userID: string): Promise<T> {
	const res = await fetch(`${backendBaseURL}${path}`, {
		headers: {
			'Content-Type': 'application/json',
			'X-User-ID': userID
		}
	});

	if (!res.ok) {
		let detail = `request failed (${res.status})`;
		try {
			const body = await res.json();
			detail = body?.error?.message ?? detail;
		} catch {
			// keep fallback error
		}
		throw new Error(detail);
	}

	return (await res.json()) as T;
}

export function shortSHA(sha?: string): string {
	if (!sha) return 'n/a';
	return sha.slice(0, 7);
}

export function prettyDate(epochSeconds?: number): string {
	if (!epochSeconds || epochSeconds <= 0) return 'n/a';
	return new Date(epochSeconds * 1000).toLocaleString();
}
