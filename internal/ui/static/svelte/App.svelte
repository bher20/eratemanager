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

  onMount(async () => {
    try {
      const resp = await fetch("/providers");
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
      const resp = await fetch(`/rates/${encodeURIComponent(provider)}/residential`);
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
      const resp = await fetch(`/refresh/${encodeURIComponent(provider)}`, { method: "POST" });
      if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
      const json = await resp.json();
      if (json.status === "ok") {
        status = "Refresh succeeded. New PDF URL: " + (json.pdf_url ?? json.pdfUrl ?? "n/a");
        statusClass = "ok";
      } else {
        status = "Refresh failed: " + (json.error ?? "unknown error");
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
    const items: { label: string; value: string }[] = [];
    if (cust != null) items.push({ label: "Customer", value: `$${cust.toFixed(2)}/mo` });
    if (energy != null) items.push({ label: "Energy", value: `${energy.toFixed(5)} $/kWh` });
    if (fuel != null) items.push({ label: "Fuel", value: `${fuel.toFixed(5)} $/kWh` });
    if (total > 0) items.push({ label: "Total", value: `${total.toFixed(5)} $/kWh` });
    return items;
  }
</script>

<style>
  .shell {
    max-width: 960px;
    margin: 1.5rem auto;
    padding: 1.25rem;
    border-radius: 0.9rem;
    background: #020617;
    border: 1px solid #1f2937;
    box-shadow: 0 18px 35px rgba(0,0,0,0.7);
    font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
    color: #e5e7eb;
  }
  h1 {
    margin-top: 0;
    font-size: 1.2rem;
  }
  label {
    display: block;
    font-size: 0.8rem;
    text-transform: uppercase;
    opacity: 0.7;
    margin-bottom: 0.25rem;
  }
  select, button {
    font: inherit;
    border-radius: 999px;
    border: 1px solid #374151;
    padding: 0.35rem 0.7rem;
    background: #020617;
    color: inherit;
    outline: none;
  }
  select:focus, button:focus {
    border-color: #3b82f6;
  }
  button {
    cursor: pointer;
  }
  button + button {
    margin-left: 0.5rem;
  }
  pre {
    margin-top: 0.8rem;
    padding: 0.7rem 0.8rem;
    border-radius: 0.6rem;
    background: #020617;
    border: 1px solid #1f2937;
    font-size: 0.78rem;
    max-height: 360px;
    overflow: auto;
  }
  .row {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    align-items: center;
    margin-top: 0.5rem;
  }
  .status {
    font-size: 0.8rem;
    margin-top: 0.3rem;
    min-height: 1.1em;
  }
  .status.ok { color: #22c55e; }
  .status.error { color: #f97373; }
  .summary {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
    gap: 0.7rem;
    margin-top: 0.75rem;
  }
  .summary-item {
    background: #020617;
    border-radius: 0.6rem;
    border: 1px solid #1f2937;
    padding: 0.55rem 0.6rem;
    font-size: 0.8rem;
  }
  .summary-item .label {
    opacity: 0.7;
    font-size: 0.76rem;
  }
  .summary-item .value {
    margin-top: 0.18rem;
    font-weight: 600;
  }
  .muted {
    font-size: 0.8rem;
    opacity: 0.7;
    margin-top: 0.5rem;
  }
  a {
    color: #93c5fd;
  }
</style>

<div class="shell">
  <h1>eRateManager Svelte UI</h1>
  <p class="muted">
    Svelte component for the same JSON APIs.
    <a href="../index.html">Back to dashboard</a>
  </p>

  <div>
    <label for="providerSelect">Provider</label>
    <select id="providerSelect" bind:value={provider}>
      {#each providers as p}
        <option value={p.key}>{p.name ?? p.key}</option>
      {/each}
    </select>
  </div>

  <div class="row">
    <button on:click={loadRates} disabled={loading}>Load</button>
    <button on:click={refreshProvider} disabled={loading}>Refresh PDF</button>
  </div>

  <div class={"status " + statusClass}>{status}</div>

  {#if data}
    <div class="summary">
      {#each extractSummary(data) as item}
        <div class="summary-item">
          <div class="label">{item.label}</div>
          <div class="value">{item.value}</div>
        </div>
      {/each}
    </div>
    <pre>{JSON.stringify(data, null, 2)}</pre>
  {:else}
    <pre>No data yet.</pre>
  {/if}
</div>
