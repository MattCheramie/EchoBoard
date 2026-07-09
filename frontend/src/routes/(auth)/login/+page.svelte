<script lang="ts">
  import { ApiError } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth';

  let email = '';
  let password = '';
  let submitting = false;
  let error: string | null = null;

  async function submit() {
    submitting = true;
    error = null;
    try {
      // The (auth) layout redirects to ?next / the dashboard once the store
      // reports `authenticated`, so no explicit navigation is needed here.
      await authStore.login({ email, password });
    } catch (err) {
      error = err instanceof ApiError ? err.message : 'could not sign in';
    } finally {
      submitting = false;
    }
  }
</script>

<h2 class="text-lg font-semibold">Sign in</h2>
<p class="mt-1 text-sm text-slate-500">Welcome back.</p>

<form class="mt-5 space-y-4" on:submit|preventDefault={submit}>
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
      autocomplete="current-password"
    />
  </div>
  {#if error}
    <p class="text-sm text-red-600 dark:text-red-400">{error}</p>
  {/if}
  <button class="btn-primary w-full" type="submit" disabled={submitting}>
    {submitting ? 'Signing in…' : 'Sign in'}
  </button>
</form>

<p class="mt-5 text-center text-sm text-slate-500">
  Have an invite? <a class="text-brand-600 hover:underline dark:text-brand-400" href="/redeem">Redeem it</a>
</p>
<p class="mt-2 text-center text-xs text-slate-400">
  First time here? <a class="hover:underline" href="/setup">Set up EchoBoard</a>
</p>
