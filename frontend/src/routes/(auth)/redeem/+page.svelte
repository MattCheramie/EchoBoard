<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { ApiError } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth';

  let token = '';
  let email = '';
  let name = '';
  let password = '';
  let submitting = false;
  let error: string | null = null;

  onMount(() => {
    // Invite links carry the token as ?token=…
    token = $page.url.searchParams.get('token') ?? '';
  });

  async function submit() {
    submitting = true;
    error = null;
    try {
      // On success the server auto-logs-in the new account; the (auth) layout
      // then redirects to the dashboard.
      await authStore.redeem({ token, email, name, password });
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'could not redeem invite';
    } finally {
      submitting = false;
    }
  }
</script>

<h2 class="text-lg font-semibold">Accept your invite</h2>
<p class="mt-1 text-sm text-slate-500">Create your EchoBoard account.</p>

<form class="mt-5 space-y-4" on:submit|preventDefault={submit}>
  <div>
    <label class="field-label" for="token">Invite token</label>
    <input id="token" class="field-input font-mono text-xs" bind:value={token} required />
  </div>
  <div>
    <label class="field-label" for="name">Name</label>
    <input id="name" class="field-input" bind:value={name} required autocomplete="name" />
  </div>
  <div>
    <label class="field-label" for="email">Email</label>
    <input
      id="email"
      class="field-input"
      type="email"
      bind:value={email}
      required
      autocomplete="username"
    />
  </div>
  <div>
    <label class="field-label" for="password">Password</label>
    <input
      id="password"
      class="field-input"
      type="password"
      bind:value={password}
      required
      autocomplete="new-password"
    />
  </div>
  {#if error}
    <p class="text-sm text-red-600 dark:text-red-400">{error}</p>
  {/if}
  <button class="btn-primary w-full" type="submit" disabled={submitting}>
    {submitting ? 'Creating account…' : 'Create account'}
  </button>
</form>

<p class="mt-5 text-center text-sm text-slate-500">
  Already have an account? <a class="text-brand-600 hover:underline dark:text-brand-400" href="/login">Sign in</a>
</p>
