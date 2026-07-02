<script lang="ts">
  import AppShell from '$lib/components/AppShell.svelte';
  export let title: string;
  export let tier: string;
  export let icon = '🚧';
</script>

<AppShell>
  <div class="mx-auto max-w-lg py-16 text-center">
    <div class="text-5xl">{icon}</div>
    <h1 class="mt-4 text-2xl font-bold">{title}</h1>
    <p class="mt-2 text-slate-500 dark:text-slate-400">
      This area arrives in {tier}. See <code>ROADMAP.md</code> for the plan.
    </p>
  </div>
</AppShell>
