<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { auth } from '$lib/stores/auth';
  import { ApiError } from '$lib/api';

  let token = '';
  let email = '';
  let name = '';
  let password = '';
  let submitting = false;
  let error = '';

  onMount(() => {
    token = $page.url.searchParams.get('token') ?? '';
  });

  async function handleSubmit() {
    submitting = true;
    error = '';
    try {
      await auth.redeem(token, email, name, password);
      goto('/');
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'Could not redeem invite';
    } finally {
      submitting = false;
    }
  }
</script>

<div class="flex min-h-screen items-center justify-center px-4">
  <div class="w-full max-w-sm">
    <div class="mb-6 text-center">
      <div class="text-3xl font-bold">EchoBoard <span aria-hidden="true">📣</span></div>
      <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">Accept your invite and create your account</p>
    </div>
    <form class="card space-y-4" on:submit|preventDefault={handleSubmit}>
      <div>
        <label class="label" for="token">Invite token</label>
        <input id="token" class="field" type="text" bind:value={token} required />
      </div>
      <div>
        <label class="label" for="name">Your name</label>
        <input id="name" class="field" type="text" autocomplete="name" bind:value={name} required />
      </div>
      <div>
        <label class="label" for="email">Email</label>
        <input id="email" class="field" type="email" autocomplete="username" bind:value={email} required />
        <p class="mt-1 text-xs text-slate-400">If your invite was pre-assigned an email, that one is used.</p>
      </div>
      <div>
        <label class="label" for="password">Password</label>
        <input id="password" class="field" type="password" autocomplete="new-password" bind:value={password} required minlength="8" />
      </div>
      {#if error}
        <p class="text-sm text-red-600">{error}</p>
      {/if}
      <button class="btn-primary w-full" type="submit" disabled={submitting}>
        {submitting ? 'Creating account…' : 'Create account'}
      </button>
      <p class="text-center text-xs text-slate-400">
        Already have an account? <a class="text-brand-600 hover:underline" href="/login">Sign in</a>.
      </p>
    </form>
  </div>
</div>
