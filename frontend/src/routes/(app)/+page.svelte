<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiError, api } from '$lib/api/client';
  import { authStore } from '$lib/stores/auth';
  import type { Invite, Role, User } from '$lib/api/types';

  let users: User[] = [];
  let usersError: string | null = null;
  let loadingUsers = false;

  // Invite form state.
  let inviteEmail = '';
  let inviteRole: Role = 'member';
  let inviteTtl = 168; // 7 days
  let inviting = false;
  let inviteError: string | null = null;
  let lastInvite: Invite | null = null;

  $: isAdmin = $authStore.user?.role === 'admin';

  onMount(() => {
    if (isAdmin) void loadUsers();
  });

  async function loadUsers() {
    loadingUsers = true;
    usersError = null;
    try {
      users = await api.listUsers();
    } catch (err) {
      usersError = err instanceof ApiError ? err.message : 'failed to load users';
    } finally {
      loadingUsers = false;
    }
  }

  async function createInvite() {
    inviting = true;
    inviteError = null;
    lastInvite = null;
    try {
      lastInvite = await api.createInvite({
        email: inviteEmail.trim() || undefined,
        role: inviteRole,
        ttlHours: inviteTtl
      });
      inviteEmail = '';
    } catch (err) {
      inviteError = err instanceof ApiError ? err.message : 'failed to create invite';
    } finally {
      inviting = false;
    }
  }

  function inviteLink(token: string): string {
    return `${window.location.origin}/redeem?token=${encodeURIComponent(token)}`;
  }
</script>

<div class="space-y-8">
  <section>
    <h1 class="text-2xl font-semibold">Welcome, {$authStore.user?.name}</h1>
    <p class="mt-1 text-slate-500">
      Signed in as {$authStore.user?.email} · <span class="capitalize">{$authStore.user?.role}</span>
    </p>
  </section>

  {#if isAdmin}
    <section class="grid gap-6 lg:grid-cols-2">
      <div class="card p-5">
        <h2 class="text-lg font-semibold">Invite a teammate</h2>
        <p class="mt-1 text-sm text-slate-500">
          EchoBoard is invite-only. Generate a time-limited link to onboard a new member.
        </p>
        <form class="mt-4 space-y-4" on:submit|preventDefault={createInvite}>
          <div>
            <label class="field-label" for="invite-email">Email (optional)</label>
            <input
              id="invite-email"
              class="field-input"
              type="email"
              bind:value={inviteEmail}
              placeholder="teammate@example.com"
              autocomplete="off"
            />
          </div>
          <div class="grid grid-cols-2 gap-4">
            <div>
              <label class="field-label" for="invite-role">Role</label>
              <select id="invite-role" class="field-input" bind:value={inviteRole}>
                <option value="member">Member</option>
                <option value="admin">Admin</option>
              </select>
            </div>
            <div>
              <label class="field-label" for="invite-ttl">Valid for (hours)</label>
              <input
                id="invite-ttl"
                class="field-input"
                type="number"
                min="1"
                bind:value={inviteTtl}
              />
            </div>
          </div>
          {#if inviteError}
            <p class="text-sm text-red-600 dark:text-red-400">{inviteError}</p>
          {/if}
          <button class="btn-primary" type="submit" disabled={inviting}>
            {inviting ? 'Creating…' : 'Create invite'}
          </button>
        </form>

        {#if lastInvite}
          <div
            class="mt-4 rounded-lg border border-brand-200 bg-brand-50 p-3 text-sm
              dark:border-brand-800 dark:bg-brand-900/30"
          >
            <p class="font-medium">Invite link (valid until {new Date(lastInvite.expiresAt).toLocaleString()}):</p>
            <code class="mt-1 block break-all text-brand-700 dark:text-brand-300">
              {inviteLink(lastInvite.token)}
            </code>
          </div>
        {/if}
      </div>

      <div class="card p-5">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold">Team</h2>
          <button class="btn-ghost text-sm" on:click={loadUsers} disabled={loadingUsers}>
            {loadingUsers ? 'Refreshing…' : 'Refresh'}
          </button>
        </div>
        {#if usersError}
          <p class="mt-4 text-sm text-red-600 dark:text-red-400">{usersError}</p>
        {:else if users.length === 0 && !loadingUsers}
          <p class="mt-4 text-sm text-slate-500">No members yet.</p>
        {:else}
          <ul class="mt-4 divide-y divide-slate-100 dark:divide-slate-800">
            {#each users as u (u.id)}
              <li class="flex items-center justify-between py-2">
                <div>
                  <p class="font-medium">{u.name}</p>
                  <p class="text-sm text-slate-500">{u.email}</p>
                </div>
                <span
                  class="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium
                    capitalize text-slate-600 dark:bg-slate-800 dark:text-slate-300"
                >
                  {u.role}
                </span>
              </li>
            {/each}
          </ul>
        {/if}
      </div>
    </section>
  {:else}
    <section class="card p-5">
      <p class="text-slate-500">
        Your workspace is ready. Content planning, the unified inbox, and analytics
        arrive in the next roadmap tiers.
      </p>
    </section>
  {/if}
</div>
