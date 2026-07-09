<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { authStore } from '$lib/stores/auth';
  import ThemeToggle from '$lib/components/ThemeToggle.svelte';

  // The root layout only renders children once auth has resolved, so here the
  // status is either `authenticated` or `anonymous`. Bounce anonymous visitors
  // to the sign-in screen, preserving where they were headed.
  $: if ($authStore.status === 'anonymous') {
    const dest = $page.url.pathname + $page.url.search;
    goto(`/login?next=${encodeURIComponent(dest)}`, { replaceState: true });
  }

  const nav = [{ href: '/', label: 'Dashboard' }];

  async function signOut() {
    await authStore.logout();
    goto('/login', { replaceState: true });
  }
</script>

{#if $authStore.status === 'authenticated'}
  <div class="min-h-screen">
    <header
      class="sticky top-0 z-10 border-b border-slate-200 bg-white/80 backdrop-blur
        dark:border-slate-800 dark:bg-slate-950/80"
    >
      <div class="mx-auto flex max-w-6xl items-center gap-6 px-4 py-3">
        <a href="/" class="flex items-center gap-2 font-semibold">
          <span aria-hidden="true">📣</span> EchoBoard
        </a>
        <nav class="flex items-center gap-1 text-sm">
          {#each nav as item}
            <a
              href={item.href}
              class="rounded-lg px-3 py-1.5 transition hover:bg-slate-100
                dark:hover:bg-slate-800"
              class:font-semibold={$page.url.pathname === item.href}
              aria-current={$page.url.pathname === item.href ? 'page' : undefined}
            >
              {item.label}
            </a>
          {/each}
        </nav>
        <div class="ml-auto flex items-center gap-3">
          <ThemeToggle />
          <span class="hidden text-sm text-slate-500 sm:inline">
            {$authStore.user?.name}
          </span>
          <button class="btn-ghost" on:click={signOut}>Sign out</button>
        </div>
      </div>
    </header>

    <main class="mx-auto max-w-6xl px-4 py-8">
      <slot />
    </main>
  </div>
{/if}
