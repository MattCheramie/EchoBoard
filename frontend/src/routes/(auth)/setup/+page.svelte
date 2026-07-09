<script lang="ts">
  // The master admin is created from the server terminal, not the browser:
  // EchoBoard is invite-only by design and the first-run bootstrap runs the
  // `--setup` CLI flow (see backend/cmd/echoboard). This screen explains that,
  // so a fresh operator isn't left looking for a "register" button that would
  // undermine the invite-only security model.
</script>

<h2 class="text-lg font-semibold">Set up EchoBoard</h2>
<p class="mt-1 text-sm text-slate-500">
  For security, the first administrator is created on the server, not from the browser.
</p>

<ol class="mt-5 space-y-3 text-sm text-slate-600 dark:text-slate-300">
  <li class="flex gap-3">
    <span class="font-semibold text-brand-600 dark:text-brand-400">1.</span>
    <div>
      On the machine running EchoBoard, bootstrap the master admin:
      <pre class="mt-2 overflow-x-auto rounded-lg bg-slate-900 p-3 text-xs text-slate-100"><code>echoboard --setup</code></pre>
    </div>
  </li>
  <li class="flex gap-3">
    <span class="font-semibold text-brand-600 dark:text-brand-400">2.</span>
    <div>Follow the prompts to set the admin email, name, and password.</div>
  </li>
  <li class="flex gap-3">
    <span class="font-semibold text-brand-600 dark:text-brand-400">3.</span>
    <div>
      Then <a class="text-brand-600 hover:underline dark:text-brand-400" href="/login">sign in</a>
      and invite your team from the dashboard.
    </div>
  </li>
</ol>
