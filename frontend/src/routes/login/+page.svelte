<script lang="ts">
  import { goto } from '$app/navigation';
  import { auth } from '$lib/stores/auth';
  import { ApiError } from '$lib/api';

  let email = '';
  let password = '';
  let submitting = false;
  let error = '';

  async function handleSubmit() {
    submitting = true;
    error = '';
    try {
      await auth.login(email, password);
      goto('/');
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'Sign in failed';
    } finally {
      submitting = false;
    }
  }
</script>

<div class="flex min-h-screen items-center justify-center px-4">
  <div class="w-full max-w-sm">
    <div class="mb-6 text-center">
      <div class="text-3xl font-bold">EchoBoard <span aria-hidden="true">📣</span></div>
      <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">Sign in to your workspace</p>
    </div>
    <form class="card space-y-4" on:submit|preventDefault={handleSubmit}>
      <div>
        <label class="label" for="email">Email</label>
        <input id="email" class="field" type="email" autocomplete="username" bind:value={email} required />
      </div>
      <div>
        <label class="label" for="password">Password</label>
        <input id="password" class="field" type="password" autocomplete="current-password" bind:value={password} required />
      </div>
      {#if error}
        <p class="text-sm text-red-600">{error}</p>
      {/if}
      <button class="btn-primary w-full" type="submit" disabled={submitting}>
        {submitting ? 'Signing in…' : 'Sign in'}
      </button>
      <p class="text-center text-xs text-slate-400">
        Have an invite? <a class="text-brand-600 hover:underline" href="/redeem">Redeem it here</a>.
      </p>
    </form>
  </div>
</div>
