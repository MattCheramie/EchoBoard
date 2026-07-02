<script lang="ts">
  import '../app.css';
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import { goto } from '$app/navigation';
  import { auth, authReady, isAuthenticated } from '$lib/stores/auth';
  import { initTheme } from '$lib/stores/theme';

  // Routes reachable without a session.
  const publicRoutes = ['/login', '/redeem'];

  onMount(() => {
    initTheme();
    auth.refresh();
  });

  $: path = $page.url.pathname;
  $: isPublic = publicRoutes.includes(path);

  // Once auth state is known, keep the user on the right side of the fence.
  $: if ($authReady) {
    if (!$isAuthenticated && !isPublic) {
      goto('/login');
    } else if ($isAuthenticated && isPublic) {
      goto('/');
    }
  }
</script>

{#if !$authReady}
  <div class="flex min-h-screen items-center justify-center text-slate-400">
    <span>Loading…</span>
  </div>
{:else}
  <slot />
{/if}
