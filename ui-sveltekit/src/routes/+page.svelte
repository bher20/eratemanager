<script lang="ts">
  import { onMount } from "svelte";

  type Provider = {
    key: string;
    name?: string;
    landingUrl?: string;
    defaultPdfPath?: string;
  };

  let providers: Provider[] = [];
  let provider = "";
  let status = "";
  let statusClass = "";
  let loading = false;
  let data: any = null;

  let usage = 1000;

  const API_BASE: string =
    (import.meta as any).env?.VITE_ERATEMANAGER_API_BASE ?? "";

  function api(path: string): string {
    return API_BASE + path;
  }

  onMount(async () => {
    try {
      const resp = await fetch(api("/providers"));
      const json = await resp.json();
      providers = json.providers ?? json.Providers ?? [];
      if (providers.length) {
        provider = providers[0].key;
      }
    } catch (err) {
      status = "Failed to load providers: " + err;
      statusClass = "error";
    }
  });

  async function loadRates() {
    if (!provider) return;
    loading = true;
    status = `Loading rates for ${provider}…`;
    statusClass = "";
    try {
      const resp = await fetch(api(`/rates/${encodeURIComponent(provider)}/residential`));
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      data = await resp.json();
      status = "Loaded rates.";
      statusClass = "ok";
    } catch (err) {
      status = "Failed to load rates: " + err;
      statusClass = "error";
    } finally {
      loading = false;
    }
  }

  async function refreshProvider() {
    if (!provider) return;
    loading = true;
    status = `Refreshing PDF for ${provider}…`;
    statusClass = "";
    try {
      const resp = await fetch(api(`/refresh/${encodeURIComponent(provider)}`), {
        method: "POST",
      });
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      const json = await resp.json();
      if (json.status === "ok") {
        status =
          "Refresh succeeded. New PDF URL: " +
          (json.pdf_url ?? json.pdfUrl ?? "n/a");
        statusClass = "ok";
      } else {
        status =
          "Refresh failed: " + (json.error ?? "unknown error");
        statusClass = "error";
      }
    } catch (err) {
      status = "Refresh failed: " + err;
      statusClass = "error";
    } finally {
      loading = false;
    }
  }

  function extractSummary(data: any) {
    if (!data || !data.rates) return [];
    const r = data.rates;
    const rs = r.residential_standard ?? r.residentialStandard ?? {};
    const energy = rs.energy_rate_usd_per_kwh ?? rs.energyRateUSDPerKWh ?? null;
    const fuel = rs.tva_fuel_rate_usd_per_kwh ?? rs.tvaFuelRateUSDPerKWh ?? null;
    const cust = rs.customer_charge_monthly_usd ?? rs.customerChargeMonthlyUSD ?? null;
    const total = (energy || 0) + (fuel || 0);
    const items: { label: string; value: string; extra?: string }[] = [];
    if (cust != null) items.push({ label: "Customer", value: `$${cust.toFixed(2)}/mo`, extra: "Fixed" });
    if (energy != null) items.push({ label: "Energy", value: `${energy.toFixed(5)} $/kWh`, extra: "Volumetric" });
    if (fuel != null) items.push({ label: "Fuel", value: `${fuel.toFixed(5)} $/kWh`, extra: "Adjustable" });
    if (total > 0) items.push({ label: "Total", value: `${total.toFixed(5)} $/kWh`, extra: "Energy + fuel" });
    return items;
  }

  function getEffectiveRates() {
    if (!data || !data.rates) return null;
    const r = data.rates;
    const rs = r.residential_standard ?? r.residentialStandard ?? {};
    const energyRate = rs.energy_rate_usd_per_kwh ?? rs.energyRateUSDPerKWh ?? 0;
    const fuelRate = rs.tva_fuel_rate_usd_per_kwh ?? rs.tvaFuelRateUSDPerKWh ?? 0;
    const customerCharge = rs.customer_charge_monthly_usd ?? rs.customerChargeMonthlyUSD ?? 0;
    return { energyRate, fuelRate, customerCharge };
  }

  $: costStats = (() => {
    const rates = getEffectiveRates();
    if (!rates) {
      return {
        total: 0,
        customer: 0,
        energy: 0,
        fuel: 0,
      };
    }
    const customer = rates.customerCharge;
    const energy = rates.energyRate * usage;
    const fuel = rates.fuelRate * usage;
    const total = customer + energy + fuel;
    return { total, customer, energy, fuel };
  })();
</script>

<svelte:head>
  <title>eRateManager SvelteKit Dashboard</title>
</svelte:head>

<main class="min-h-screen bg-base-300 text-base-content">
  <div class="max-w-7xl mx-auto px-4 py-6 space-y-6">
    <div class="navbar bg-base-100 rounded-box shadow-lg">
      <div class="flex-1">
        <span class="btn btn-ghost normal-case text-xl">eRateManager</span>
        <span class="ml-2 badge badge-outline">SvelteKit + DaisyUI</span>
      </div>
      <div class="flex-none">
        <span class="badge badge-success badge-sm mr-2"></span>
        <span class="text-sm opacity-70">Backend online</span>
      </div>
    </div>

    <div class="grid gap-4 lg:grid-cols-3">
      <!-- Provider card -->
      <section class="card bg-base-100 shadow-xl lg:col-span-1">
        <div class="card-body space-y-4">
          <h2 class="card-title">Provider</h2>
          <p class="text-sm opacity-70">
            Select a utility and load its current residential tariff.
          </p>

          <div class="form-control w-full">
            <label class="label">
              <span class="label-text text-xs uppercase tracking-wide">Utility provider</span>
            </label>
            <select
              class="select select-bordered select-sm"
              bind:value={provider}
            >
              {#each providers as p}
                <option value={p.key}>{p.name ?? p.key}</option>
              {/each}
            </select>
          </div>

          <div class="text-xs opacity-80 space-y-1">
            {#if providers.length}
              {#each providers as p}
                {#if p.key === provider}
                  {#if p.landingUrl}
                    <div>
                      Landing page:
                      <a class="link link-primary" href={p.landingUrl} target="_blank" rel="noreferrer">
                        {p.landingUrl}
                      </a>
                    </div>
                  {/if}
                  {#if p.defaultPdfPath}
                    <div>Default PDF path: <code>{p.defaultPdfPath}</code></div>
                  {/if}
                {/if}
              {/each}
            {:else}
              <span>No providers loaded yet.</span>
            {/if}
          </div>

          <div class="card-actions justify-start">
            <button
              class="btn btn-primary btn-sm"
              on:click={loadRates}
              disabled={loading || !provider}
            >
              {#if loading}
                <span class="loading loading-spinner loading-xs"></span>
                Loading…
              {:else}
                Load rates
              {/if}
            </button>
            <button
              class="btn btn-ghost btn-sm"
              on:click={refreshProvider}
              disabled={loading || !provider}
            >
              ↻ Refresh PDF
            </button>
          </div>

          <div class:text-success={statusClass === "ok"} class:text-error={statusClass === "error"} class="text-xs min-h-[1.25rem]">
            {status}
          </div>
        </div>
      </section>

      <!-- Rates card -->
      <section class="card bg-base-100 shadow-xl lg:col-span-1">
        <div class="card-body space-y-3">
          <div class="flex items-center justify-between gap-2">
            <div>
              <h2 class="card-title">Rates</h2>
              <p class="text-xs opacity-70">
                Snapshot of key components from the current tariff.
              </p>
            </div>
            <div class="badge badge-outline">
              {#if provider}
                {provider.toUpperCase()}
              {:else}
                —
              {/if}
            </div>
          </div>

          <div class="grid grid-cols-2 gap-2">
            {#if data}
              {#each extractSummary(data) as item}
                <div class="stat bg-base-200 rounded-box">
                  <div class="stat-title text-xs">{item.label}</div>
                  <div class="stat-value text-sm">{item.value}</div>
                  {#if item.extra}
                    <div class="stat-desc text-[0.7rem] opacity-70">
                      {item.extra}
                    </div>
                  {/if}
                </div>
              {/each}
            {:else}
              <div class="col-span-2 text-xs opacity-70">
                No data yet. Choose a provider and click
                <span class="badge badge-outline badge-xs mx-1">Load rates</span>.
              </div>
            {/if}
          </div>

          <details class="collapse collapse-arrow border border-base-200 bg-base-200">
            <summary class="collapse-title text-xs font-medium">
              Raw JSON response
            </summary>
            <div class="collapse-content">
              <pre class="text-xs overflow-auto max-h-64 bg-base-300 rounded-box p-2">
{data ? JSON.stringify(data, null, 2) : "No data loaded."}
              </pre>
            </div>
          </details>
        </div>
      </section>

      <!-- Cost explorer card -->
      <section class="card bg-base-100 shadow-xl lg:col-span-1">
        <div class="card-body space-y-4">
          <h2 class="card-title">Cost explorer</h2>
          <p class="text-xs opacity-70">
            Estimate a monthly bill using the currently loaded rate structure.
          </p>

          <div class="form-control w-full">
            <label class="label">
              <span class="label-text text-xs uppercase tracking-wide">
                Monthly usage (kWh)
              </span>
            </label>
            <div class="flex items-center gap-3">
              <input
                class="input input-bordered input-xs w-24"
                type="number"
                min="0"
                step="50"
                bind:value={usage}
              />
              <span class="text-[0.7rem] opacity-70">
                Try 500, 1000, 1500…
              </span>
            </div>
            <input
              type="range"
              min="0"
              max="2500"
              step="50"
              bind:value={usage}
              class="range range-xs mt-2"
            />
          </div>

          <div class="stats stats-vertical shadow bg-base-200">
            <div class="stat">
              <div class="stat-title text-xs">Estimated bill</div>
              <div class="stat-value text-lg">
                {#if costStats.total}
                  ${costStats.total.toFixed(2)}
                {:else}
                  —
                {/if}
              </div>
              <div class="stat-desc text-[0.7rem] opacity-70">
                for approximately {usage} kWh
              </div>
            </div>

            <div class="stat">
              <div class="stat-title text-xs">Customer portion</div>
              <div class="stat-value text-sm">
                {#if costStats.customer}
                  ${costStats.customer.toFixed(2)}
                {:else}
                  —
                {/if}
              </div>
              <div class="stat-desc text-[0.7rem] opacity-70">
                {#if costStats.total}
                  {(costStats.customer / costStats.total * 100).toFixed(1)}% of bill
                {/if}
              </div>
            </div>

            <div class="stat">
              <div class="stat-title text-xs">Energy portion</div>
              <div class="stat-value text-sm">
                {#if costStats.energy}
                  ${costStats.energy.toFixed(2)}
                {:else}
                  —
                {/if}
              </div>
              <div class="stat-desc text-[0.7rem] opacity-70">
                {#if costStats.total}
                  {(costStats.energy / costStats.total * 100).toFixed(1)}% of bill
                {/if}
              </div>
            </div>

            <div class="stat">
              <div class="stat-title text-xs">Fuel portion</div>
              <div class="stat-value text-sm">
                {#if costStats.fuel}
                  ${costStats.fuel.toFixed(2)}
                {:else}
                  —
                {/if}
              </div>
              <div class="stat-desc text-[0.7rem] opacity-70">
                {#if costStats.total}
                  {(costStats.fuel / costStats.total * 100).toFixed(1)}% of bill
                {/if}
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</main>
