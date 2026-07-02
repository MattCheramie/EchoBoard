<script lang="ts">
  import { onMount } from 'svelte';
  import AppShell from '$lib/components/AppShell.svelte';
  import { api, ApiError } from '$lib/api';
  import { currentUser, isAdmin } from '$lib/stores/auth';
  import type { Invite, Role, User } from '$lib/types';

  let users: User[] = [];
  let loadingUsers = false;
  let usersError = '';

  // Invite form state.
  let inviteEmail = '';
  let inviteRole: Role = 'member';
  let creating = false;
  let createdInvite: Invite | null = null;
  let inviteError = '';

  async function loadUsers() {
    loadingUsers = true;
    usersError = '';
    try {
      users = await api.listUsers();
    } catch (err) {
      usersError = err instanceof ApiError ? err.message : 'Failed to load users';
    } finally {
      loadingUsers = false;
    }
  }

  async function createInvite() {
    creating = true;
    inviteError = '';
    createdInvite = null;
    try {
      createdInvite = await api.createInvite({
        email: inviteEmail || undefined,
        role: inviteRole
      });
      inviteEmail = '';
    } catch (err) {
      inviteError = err instanceof ApiError ? err.message : 'Failed to create invite';
    } finally {
      creating = false;
    }
  }

  function inviteLink(token: string): string {
    return `${location.origin}/redeem?token=${encodeURIComponent(token)}`;
  }

  onMount(() => {
    if ($isAdmin) loadUsers();
  });
</script>

<AppShell>
  <h1 class="text-2xl font-bold">Welcome back, {$currentUser?.name}</h1>
  <p class="mt-1 text-slate-500 dark:text-slate-400">
    This is the EchoBoard shell. Content calendar, unified inbox, and analytics arrive in later tiers.
  </p>

  <div class="mt-8 grid gap-6 md:grid-cols-3">
    {#each [{ t: 'Calendar', d: 'Plan and schedule content', e: '📅' }, { t: 'Inbox', d: 'Unified conversations', e: '💬' }, { t: 'People', d: 'Integrated CRM', e: '👥' }] as f}
      <div class="card">
        <div class="text-2xl">{f.e}</div>
        <div class="mt-2 font-semibold">{f.t}</div>
        <div class="text-sm text-slate-500 dark:text-slate-400">{f.d}</div>
      </div>
    {/each}
  </div>

  {#if $isAdmin}
    <section class="mt-10 grid gap-6 lg:grid-cols-2">
      <div class="card">
        <h2 class="text-lg font-semibold">Invite a team member</h2>
        <p class="mt-1 text-sm text-slate-500 dark:text-slate-400">
          EchoBoard is invite-only. Generate a time-limited link to onboard someone.
        </p>
        <form class="mt-4 space-y-3" on:submit|preventDefault={createInvite}>
          <div>
            <label class="label" for="inviteEmail">Email (optional)</label>
            <input id="inviteEmail" class="field" type="email" bind:value={inviteEmail} placeholder="teammate@example.com" />
          </div>
          <div>
            <label class="label" for="inviteRole">Role</label>
            <select id="inviteRole" class="field" bind:value={inviteRole}>
              <option value="member">Member</option>
              <option value="admin">Admin</option>
            </select>
          </div>
          <button class="btn-primary" type="submit" disabled={creating}>
            {creating ? 'Creating…' : 'Create invite'}
          </button>
          {#if inviteError}
            <p class="text-sm text-red-600">{inviteError}</p>
          {/if}
        </form>

        {#if createdInvite}
          <div class="mt-4 rounded-md bg-brand-50 p-3 text-sm dark:bg-slate-800">
            <div class="font-medium">Invite link (share securely):</div>
            <code class="mt-1 block break-all text-brand-700 dark:text-brand-100">
              {inviteLink(createdInvite.token)}
            </code>
          </div>
        {/if}
      </div>

      <div class="card">
        <div class="flex items-center justify-between">
          <h2 class="text-lg font-semibold">Team</h2>
          <button class="btn-ghost" on:click={loadUsers} disabled={loadingUsers}>Refresh</button>
        </div>
        {#if usersError}
          <p class="mt-3 text-sm text-red-600">{usersError}</p>
        {:else if loadingUsers}
          <p class="mt-3 text-sm text-slate-500">Loading…</p>
        {:else}
          <ul class="mt-3 divide-y divide-slate-200 dark:divide-slate-800">
            {#each users as u}
              <li class="flex items-center justify-between py-2 text-sm">
                <span>{u.name} <span class="text-slate-400">· {u.email}</span></span>
                <span class="rounded bg-slate-100 px-2 py-0.5 text-xs dark:bg-slate-800">{u.role}</span>
              </li>
            {/each}
          </ul>
        {/if}
      </div>
    </section>
  {/if}
</AppShell>
