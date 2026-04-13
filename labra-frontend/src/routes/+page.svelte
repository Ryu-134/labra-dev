<script>
    import HeroSection from '$lib/components/hero.svelte';
</script>

<HeroSection />

<section class="cta">
    <a href="/apps">Open App Dashboard</a>
</section>

<style>
    .cta {
        display: flex;
        justify-content: center;
        padding-bottom: 2rem;
    }
    .cta a {
        background: #b9c2ff;
        color: #11111b;
        border-radius: 12px;
        padding: 0.8rem 1.2rem;
        text-decoration: none;
        font-weight: 600;
    }
</style>
