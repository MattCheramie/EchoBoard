<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { authStore } from '$lib/stores/auth';
  import { theme } from '$lib/stores/theme';

  let bootError: string | null = null;

  onMount(async () => {
    theme.init();
    try {
      await authStore.refresh();
    } catch (err) {
      // A non-401 failure means the API is unreachable or erroring; the auth
      // store stays in `loading`, so show a retry rather than a blank screen.
      bootError = err instanceof Error ? err.message : 'could not reach the server';
    }
  });

  async function retry() {
    bootError = null;
    try {
      await authStore.refresh();
    } catch (err) {
      bootError = err instanceof Error ? err.message : 'could not reach the server';
    }
  }
</script>

{#if $authStore.status === 'loading' && !bootError}
  <div class="flex min-h-screen items-center justify-center">
    <div class="flex items-center gap-3 text-slate-500">
      <span
        class="h-5 w-5 animate-spin rounded-full border-2 border-slate-300 border-t-brand-600"
        aria-hidden="true"
      ></span>
      <span>Loading EchoBoard…</span>
    </div>
  </div>
{:else if bootError}
  <div class="flex min-h-screen flex-col items-center justify-center gap-4 p-8 text-center">
    <h1 class="text-2xl font-semibold">Can’t reach EchoBoard</h1>
    <p class="max-w-md text-slate-500">{bootError}</p>
    <button class="btn-primary" on:click={retry}>Try again</button>
  </div>
{:else}
  <slot />
{/if}
