<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { auth, currentUser, isAdmin } from '$lib/stores/auth';
  import { theme, toggleTheme } from '$lib/stores/theme';

  const nav = [
    { href: '/', label: 'Dashboard' },
    { href: '/calendar', label: 'Calendar' },
    { href: '/inbox', label: 'Inbox' },
    { href: '/people', label: 'People' }
  ];

  async function handleLogout() {
    await auth.logout();
    goto('/login');
  }

  $: path = $page.url.pathname;
</script>

<div class="min-h-screen">
  <header class="sticky top-0 z-10 border-b border-slate-200 bg-white/80 backdrop-blur dark:border-slate-800 dark:bg-slate-950/80">
    <div class="mx-auto flex max-w-6xl items-center justify-between px-4 py-3">
      <div class="flex items-center gap-6">
        <a href="/" class="text-lg font-bold">EchoBoard <span aria-hidden="true">📣</span></a>
        <nav class="hidden gap-1 md:flex">
          {#each nav as item}
            <a
              href={item.href}
              class="rounded-md px-3 py-1.5 text-sm font-medium transition-colors"
              class:bg-brand-50={path === item.href}
              class:text-brand-700={path === item.href}
              class:text-slate-600={path !== item.href}
              class:dark:text-slate-300={path !== item.href}
              class:hover:bg-slate-100={path !== item.href}
              class:dark:hover:bg-slate-800={path !== item.href}
            >
              {item.label}
            </a>
          {/each}
        </nav>
      </div>

      <div class="flex items-center gap-3">
        <button class="btn-ghost" on:click={toggleTheme} aria-label="Toggle theme" title="Toggle theme">
          {$theme === 'dark' ? '☀️' : '🌙'}
        </button>
        {#if $currentUser}
          <span class="hidden text-sm text-slate-500 sm:inline dark:text-slate-400">
            {$currentUser.name}{#if $isAdmin}&nbsp;· admin{/if}
          </span>
        {/if}
        <button class="btn-ghost" on:click={handleLogout}>Sign out</button>
      </div>
    </div>
  </header>

  <main class="mx-auto max-w-6xl px-4 py-8">
    <slot />
  </main>
</div>
