<script lang="ts">
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { authStore } from '$lib/stores/auth';

  // Signed-in users have no business on the auth screens — send them where they
  // were trying to go (?next=) or to the dashboard.
  $: if ($authStore.status === 'authenticated') {
    const next = $page.url.searchParams.get('next');
    goto(next && next.startsWith('/') ? next : '/', { replaceState: true });
  }
</script>

{#if $authStore.status !== 'authenticated'}
  <div class="flex min-h-screen items-center justify-center p-4">
    <div class="w-full max-w-sm">
      <div class="mb-6 text-center">
        <div class="text-3xl">📣</div>
        <h1 class="mt-2 text-xl font-semibold">EchoBoard</h1>
      </div>
      <div class="card p-6">
        <slot />
      </div>
    </div>
  </div>
{/if}
