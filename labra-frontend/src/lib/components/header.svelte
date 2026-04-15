<script>
    import GithubLoginButton from '$lib/components/githublogin.svelte';

    const appEnv = import.meta.env.PUBLIC_APP_ENV ?? 'local';
</script>

<div class="header">
    <a id="logo" href="/">Labra</a>
    <nav>
        <a href="/dashboard">Dashboard</a>
        <a href="/apps">Apps</a>
        <a href="/deploys">Deploys</a>
        <a href="/settings">Settings</a>
        <a href="/login">Login</a>
    </nav>
    <div class="meta">
        <span class="env">Env: {appEnv}</span>
        <div class="login">
            <GithubLoginButton />
        </div>
    </div>
</div>
<hr>
<slot />

<style>
    .header {
        background-color: var(--crust);
        min-height: 7vh;
        color: var(--text-color);
        width: 100%;
        display: flex;
        padding-top: 2vh;
        padding-bottom: 2vh;
        justify-content: space-between;
        align-items: center;
        gap: 1rem;
    }
    #logo {
        margin-left: 3vh;
        font-size: 5vh;
        color: var(--text-color);
        text-decoration: none;
    }
    nav {
        display: flex;
        flex-wrap: wrap;
        gap: 1rem;
        margin-right: auto;
    }
    nav a {
        color: #b8c2ff;
        text-decoration: none;
        font-size: 1.1rem;
    }
    .meta {
        margin-right: 3vh;
        display: flex;
        align-items: center;
        gap: 0.8rem;
    }
    .env {
        color: #a3b1db;
        font-size: 0.9rem;
        border: 1px solid #3f4565;
        border-radius: 8px;
        padding: 0.3rem 0.5rem;
    }
    .login {
        display: flex;
        justify-content: center;
    }
    hr {
        height: 0.2vh;
        border-width: 0;
        color: var(--hr-color);
        background-color: var(--hr-color);
    }
</style>
