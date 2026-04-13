<script>
    import GithubLoginButton from '$lib/components/githublogin.svelte';
</script>

<div class="header">
    <a id="logo" href="/">Labra</a>
    <nav>
        <a href="/apps">Apps</a>
    </nav>
    <div class="login">
        <GithubLoginButton />
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
        gap: 1rem;
        margin-right: auto;
    }
    nav a {
        color: #b8c2ff;
        text-decoration: none;
        font-size: 1.1rem;
    }
    .login {
        margin-right: 3vh;
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
