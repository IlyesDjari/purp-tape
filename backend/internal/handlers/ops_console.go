package handlers

import (
	"encoding/json"
	"net/http"
)

type opsEndpoint struct {
	Method string `json:"method"`
	Path   string `json:"path"`
	Scope  string `json:"scope"`
	Notes  string `json:"notes"`
}

// OpsConsole renders a lightweight operations frontend for smoke testing and API exploration.
func OpsConsole(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(`<!doctype html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>PurpTape Ops Console</title>
  <style>
    :root {
      --bg: #0b1020;
      --panel: #131a2e;
      --muted: #90a4c2;
      --text: #e8efff;
      --ok: #19c37d;
      --warn: #ffb020;
      --err: #ff5d5d;
      --accent: #4f8cff;
      --border: #26304d;
    }
    * { box-sizing: border-box; }
    body {
      margin: 0;
      font-family: Inter, ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, Helvetica, Arial;
      background: linear-gradient(180deg, #0b1020 0%, #0a0f1c 100%);
      color: var(--text);
    }
    .wrap {
      max-width: 1160px;
      margin: 0 auto;
      padding: 28px 18px 48px;
    }
    h1 { margin: 0 0 8px; font-size: 28px; }
    .subtitle { color: var(--muted); margin-bottom: 20px; }
    .grid {
      display: grid;
      grid-template-columns: repeat(12, 1fr);
      gap: 14px;
    }
    .card {
      background: var(--panel);
      border: 1px solid var(--border);
      border-radius: 12px;
      padding: 14px;
    }
    .span-4 { grid-column: span 4; }
    .span-6 { grid-column: span 6; }
    .span-8 { grid-column: span 8; }
    .span-12 { grid-column: span 12; }
    @media (max-width: 940px) {
      .span-4, .span-6, .span-8 { grid-column: span 12; }
    }
    .k {
      font-size: 12px;
      color: var(--muted);
      text-transform: uppercase;
      letter-spacing: .08em;
      margin-bottom: 6px;
    }
    .v { font-size: 22px; font-weight: 700; }
    .row { display: flex; gap: 10px; align-items: center; flex-wrap: wrap; }
    .badge {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      border-radius: 999px;
      border: 1px solid var(--border);
      padding: 4px 10px;
      font-size: 12px;
      color: var(--muted);
      background: #0d1427;
    }
    .dot { width: 8px; height: 8px; border-radius: 999px; background: var(--warn); }
    .dot.ok { background: var(--ok); }
    .dot.err { background: var(--err); }
    .dot.warn { background: var(--warn); }
    input, textarea, select, button {
      background: #0d1427;
      border: 1px solid var(--border);
      color: var(--text);
      border-radius: 10px;
      padding: 10px 12px;
      font: inherit;
    }
    input, select, textarea { width: 100%; }
    textarea { min-height: 120px; resize: vertical; }
    button {
      cursor: pointer;
      background: linear-gradient(180deg, #4f8cff, #3d77e6);
      border: none;
      font-weight: 600;
    }
    table {
      width: 100%;
      border-collapse: collapse;
      font-size: 14px;
    }
    th, td {
      text-align: left;
      padding: 8px 6px;
      border-bottom: 1px solid #1d2743;
    }
    th { color: var(--muted); font-weight: 600; }
    pre {
      margin: 0;
      background: #0a1020;
      border: 1px solid var(--border);
      border-radius: 10px;
      padding: 10px;
      white-space: pre-wrap;
      word-break: break-word;
      max-height: 360px;
      overflow: auto;
      font-size: 13px;
    }
    .small { font-size: 12px; color: var(--muted); }
  </style>
</head>
<body>
  <div class="wrap">
    <h1>PurpTape Founder Dashboard</h1>
    <div class="subtitle">Real-time analytics, costs, and system health for PurpTape founders.</div>

    <!-- TAB SELECTOR -->
    <div style="margin-bottom: 20px;">
      <button id="tabDashboard" class="tabBtn" style="background: linear-gradient(180deg, #4f8cff, #3d77e6); border: none; padding: 8px 16px; border-radius: 6px; margin-right: 8px; color: var(--text); cursor: pointer;">Dashboard</button>
      <button id="tabOps" class="tabBtn" style="background: var(--panel); border: 1px solid var(--border); padding: 8px 16px; border-radius: 6px; margin-right: 8px; color: var(--text); cursor: pointer;">Ops Console</button>
    </div>

    <!-- DASHBOARD VIEW -->
    <div id="dashboardView" class="grid">
      <!-- COSTS SECTION -->
      <section class="card span-4">
        <div class="k">💰 Current Month Cost</div>
        <div class="v" id="currentCost">$0.00</div>
        <div class="small" id="costStatus" style="margin-top: 8px;">Loading...</div>
      </section>

      <section class="card span-4">
        <div class="k">📊 Budget Utilization</div>
        <div class="v" id="budgetPercent">0%</div>
        <div class="small" id="budgetInfo" style="margin-top: 8px;">of $500 budget</div>
      </section>

      <section class="card span-4">
        <div class="k">🏥 System Health</div>
        <div class="row"><span class="badge"><span class="dot" id="healthDot"></span><span id="healthText">Checking...</span></span></div>
      </section>

      <!-- COST BREAKDOWN -->
      <section class="card span-4">
        <div class="k">📁 Storage Costs</div>
        <div class="v" id="storageCost">$0.00</div>
      </section>

      <section class="card span-4">
        <div class="k">⚡ API Costs</div>
        <div class="v" id="apiCost">$0.00</div>
      </section>

      <section class="card span-4">
        <div class="k">🔄 Transfer Costs</div>
        <div class="v" id="transferCost">$0.00</div>
      </section>

      <!-- HIGH COST PROJECTS -->
      <section class="card span-12">
        <div class="k">⚠️ High Cost Projects</div>
        <table>
          <thead><tr><th>Project</th><th>Cost</th><th>% of Total</th><th>Optimization Tip</th></tr></thead>
          <tbody id="highCostBody"><tr><td colspan="4">Loading...</td></tr></tbody>
        </table>
      </section>

      <!-- ENGAGEMENT -->
      <section class="card span-6">
        <div class="k">🎵 Total Plays (This Month)</div>
        <div class="v" id="totalPlays">0</div>
      </section>

      <section class="card span-6">
        <div class="k">👥 Unique Listeners</div>
        <div class="v" id="uniqueListeners">0</div>
      </section>

      <!-- SYSTEM DETAILS -->
      <section class="card span-12">
        <div class="k">🖥️ System Metrics</div>
        <table>
          <tbody>
            <tr><td>DB Connections</td><td id="dbConnections">-</td></tr>
            <tr><td>Memory Usage</td><td id="memoryUsage">-</td></tr>
            <tr><td>Uptime</td><td id="uptime">-</td></tr>
            <tr><td>Last Refresh</td><td id="lastRefresh" style="color: var(--muted);">-</td></tr>
          </tbody>
        </table>
      </section>

      <!-- ACTIONS -->
      <section class="card span-12">
        <div class="k">Actions</div>
        <button id="refreshDash" style="background: linear-gradient(180deg, #4f8cff, #3d77e6); border: none; padding: 10px 16px; border-radius: 6px; color: var(--text); cursor: pointer;">Refresh Dashboard</button>
      </section>
    </div>

    <!-- OPS CONSOLE VIEW (ORIGINAL) -->
    <div id="opsView" style="display: none;" class="grid">
      <section class="card span-4">
        <div class="k">Service</div>
        <div class="v" id="serviceName">purptape-api</div>
        <div class="small" id="baseUrl"></div>
      </section>

      <section class="card span-4">
        <div class="k">Overall Health</div>
        <div class="row"><span class="badge"><span class="dot" id="healthDot2"></span><span id="healthText2">Checking...</span></span></div>
      </section>

      <section class="card span-4">
        <div class="k">Readiness</div>
        <div class="row"><span class="badge"><span class="dot" id="readyDot"></span><span id="readyText">Checking...</span></span></div>
      </section>

      <section class="card span-6">
        <div class="k">Quick Checks</div>
        <div class="row" style="margin-bottom:10px">
          <button id="refreshBtn">Refresh checks</button>
          <button id="sampleBtn">Run sample smoke</button>
        </div>
        <pre id="quickOut">No checks run yet.</pre>
      </section>

      <section class="card span-6">
        <div class="k">Auth Token</div>
        <input id="token" type="password" placeholder="Paste Bearer token (optional for protected routes)" />
        <div class="small" style="margin-top:8px">Token is kept in this browser tab only and never persisted.</div>
      </section>

      <section class="card span-12">
        <div class="k">Endpoint Catalog</div>
        <table>
          <thead><tr><th>Method</th><th>Path</th><th>Scope</th><th>Notes</th></tr></thead>
          <tbody id="endpointsBody"></tbody>
        </table>
      </section>

      <section class="card span-12">
        <div class="k">Request Tester</div>
        <div class="grid">
          <div class="span-4">
            <label class="small">Method</label>
            <select id="method">
              <option>GET</option>
              <option>POST</option>
              <option>PATCH</option>
              <option>DELETE</option>
            </select>
          </div>
          <div class="span-8">
            <label class="small">Path</label>
            <input id="path" value="/health" />
          </div>
          <div class="span-12">
            <label class="small">JSON Body (for POST/PATCH)</label>
            <textarea id="body" placeholder='{"key":"value"}'></textarea>
          </div>
          <div class="span-12 row">
            <button id="sendBtn">Send Request</button>
          </div>
        </div>
        <div style="margin-top:10px" class="small">Response</div>
        <pre id="resp">-</pre>
      </section>
    </div>
  </div>

  <script>
    const baseUrl = window.location.origin;
    document.getElementById('baseUrl').textContent = baseUrl;

    // TAB MANAGEMENT
    function switchTab(tab) {
      const dashView = document.getElementById('dashboardView');
      const opsView = document.getElementById('opsView');
      const dashBtn = document.getElementById('tabDashboard');
      const opsBtn = document.getElementById('tabOps');
      
      if (tab === 'dashboard') {
        dashView.style.display = 'grid';
        opsView.style.display = 'none';
        dashBtn.style.background = 'linear-gradient(180deg, #4f8cff, #3d77e6)';
        dashBtn.style.borderColor = 'transparent';
        opsBtn.style.background = 'var(--panel)';
        opsBtn.style.borderColor = '1px solid var(--border)';
        refreshDashboard();
      } else {
        dashView.style.display = 'none';
        opsView.style.display = 'grid';
        dashBtn.style.background = 'var(--panel)';
        dashBtn.style.borderColor = '1px solid var(--border)';
        opsBtn.style.background = 'linear-gradient(180deg, #4f8cff, #3d77e6)';
        opsBtn.style.borderColor = 'transparent';
        refreshChecks();
        loadEndpoints();
      }
    }

    document.getElementById('tabDashboard').addEventListener('click', () => switchTab('dashboard'));
    document.getElementById('tabOps').addEventListener('click', () => switchTab('ops'));
    document.getElementById('refreshDash').addEventListener('click', refreshDashboard);

    // DASHBOARD FUNCTIONS
    async function jsonFetch(path, options = {}) {
      const token = document.getElementById('token').value.trim();
      const headers = Object.assign({ 'Accept': 'application/json' }, options.headers || {});
      if (token) headers['Authorization'] = 'Bearer ' + token;
      const res = await fetch(baseUrl + path, { ...options, headers });
      const text = await res.text();
      let data;
      try { data = JSON.parse(text); } catch { data = text; }
      return { status: res.status, ok: res.ok, data };
    }

    function setDot(id, kind) {
      const el = document.getElementById(id);
      if (!el) return;
      el.classList.remove('ok', 'warn', 'err');
      if (kind) el.classList.add(kind);
    }

    async function refreshDashboard() {
      const token = document.getElementById('token').value.trim();
      if (!token) {
        document.getElementById('currentCost').textContent = 'Please add JWT token above';
        return;
      }

      try {
        const res = await jsonFetch('/api/founder/dashboard');
        if (!res.ok) {
          document.getElementById('currentCost').textContent = 'Error: ' + String(res.status);
          setDot('healthDot', 'err');
          return;
        }

        const dash = res.data;
        
        // COSTS
        document.getElementById('currentCost').textContent = '$' + dash.costs.current_month_usd.toFixed(2);
        document.getElementById('costStatus').textContent = dash.costs.utilization_percent.toFixed(1) + '% of $' + dash.costs.budget_usd.toFixed(0) + ' budget';
        document.getElementById('budgetPercent').textContent = dash.costs.utilization_percent.toFixed(0) + '%';
        document.getElementById('storageCost').textContent = '$' + dash.costs.storage_cost_usd.toFixed(2);
        document.getElementById('apiCost').textContent = '$' + dash.costs.api_cost_usd.toFixed(2);
        document.getElementById('transferCost').textContent = '$' + dash.costs.transfer_cost_usd.toFixed(2);

        // HEALTH
        setDot('healthDot', dash.system.status === 'healthy' ? 'ok' : 'warn');
        document.getElementById('healthText').textContent = dash.system.status === 'healthy' ? 'Healthy' : 'Warning';

        // HIGH COST PROJECTS
        const body = document.getElementById('highCostBody');
        body.innerHTML = '';
        if (dash.projects.high_cost_projects && dash.projects.high_cost_projects.length > 0) {
          dash.projects.high_cost_projects.forEach(proj => {
            const tr = document.createElement('tr');
            tr.innerHTML = '<td>' + proj.name + '</td><td>$' + proj.cost_usd.toFixed(2) + '</td><td>' + proj.cost_percent.toFixed(1) + '%</td><td style="font-size: 12px; color: var(--muted);">' + proj.optimization_tip + '</td>';
            body.appendChild(tr);
          });
        } else {
          body.innerHTML = '<tr><td colspan="4">No cost data available yet</td></tr>';
        }

        // ENGAGEMENT
        document.getElementById('totalPlays').textContent = String(dash.engagement.total_plays_month || 0);
        document.getElementById('uniqueListeners').textContent = String(dash.engagement.unique_listeners_month || 0);

        // SYSTEM
        document.getElementById('dbConnections').textContent = dash.system.db_connections_used + ' / ' + dash.system.db_connections_max;
        document.getElementById('memoryUsage').textContent = dash.system.memory_mb + ' MB';
        document.getElementById('uptime').textContent = dash.system.uptime_hours + ' hours';
        document.getElementById('lastRefresh').textContent = 'Just now';

      } catch (e) {
        document.getElementById('currentCost').textContent = 'Error: ' + String(e);
        setDot('healthDot', 'err');
      }
    }

    async function refreshChecks() {
      const out = [];
      try {
        const health = await jsonFetch('/health');
        setDot('healthDot2', health.ok ? 'ok' : 'err');
        document.getElementById('healthText2').textContent = String(health.status) + ' ' + (health.ok ? 'OK' : 'FAIL');
        out.push(['GET /health', health.status, health.data]);
      } catch (e) {
        setDot('healthDot2', 'err');
        document.getElementById('healthText2').textContent = 'error';
      }

      try {
        const ready = await jsonFetch('/readiness');
        setDot('readyDot', ready.ok ? 'ok' : 'warn');
        document.getElementById('readyText').textContent = String(ready.status) + ' ' + (ready.ok ? 'READY' : 'NOT READY');
        out.push(['GET /readiness', ready.status, ready.data]);
      } catch {
        setDot('readyDot', 'err');
        document.getElementById('readyText').textContent = 'error';
      }

      document.getElementById('quickOut').textContent = out.map(v => String(v[0]) + ' -> ' + String(v[1]) + '\n' + JSON.stringify(v[2], null, 2)).join('\n\n');
    }

    async function runSampleSmoke() {
      const checks = ['/health', '/health/deep', '/pricing/tiers', '/discover/trending'];
      const lines = [];
      for (const path of checks) {
        try {
          const r = await jsonFetch(path);
          lines.push(path.padEnd(20) + ' ' + String(r.status).padStart(3) + ' ' + (r.ok ? 'OK' : 'FAIL'));
        } catch (e) {
          lines.push(path.padEnd(20) + ' ERR ' + String(e));
        }
      }
      document.getElementById('quickOut').textContent = lines.join('\n');
    }

    async function loadEndpoints() {
      const body = document.getElementById('endpointsBody');
      body.innerHTML = '<tr><td colspan="4">Loading...</td></tr>';
      try {
        const res = await jsonFetch('/ops/endpoints');
        if (!res.ok || !Array.isArray(res.data)) {
          body.innerHTML = '<tr><td colspan="4">Failed (' + String(res.status) + ')</td></tr>';
          return;
        }
        body.innerHTML = '';
        for (const ep of res.data) {
          const tr = document.createElement('tr');
          tr.innerHTML = '<td>' + String(ep.method) + '</td><td>' + String(ep.path) + '</td><td>' + String(ep.scope) + '</td><td>' + String(ep.notes || '') + '</td>';
          body.appendChild(tr);
        }
      } catch (e) {
        body.innerHTML = '<tr><td colspan="4">Error: ' + String(e) + '</td></tr>';
      }
    }

    async function sendRequest() {
      const method = document.getElementById('method').value;
      const path = document.getElementById('path').value.trim();
      const bodyRaw = document.getElementById('body').value.trim();
      const opts = { method };
      if ((method === 'POST' || method === 'PATCH') && bodyRaw) {
        opts.headers = { 'Content-Type': 'application/json' };
        opts.body = bodyRaw;
      }
      try {
        const res = await jsonFetch(path, opts);
        document.getElementById('resp').textContent = 'HTTP ' + String(res.status) + '\n\n' + (typeof res.data === 'string' ? res.data : JSON.stringify(res.data, null, 2));
      } catch (e) {
        document.getElementById('resp').textContent = String(e);
      }
    }

    // EVENT LISTENERS
    document.getElementById('refreshBtn').addEventListener('click', refreshChecks);
    document.getElementById('sampleBtn').addEventListener('click', runSampleSmoke);
    document.getElementById('sendBtn').addEventListener('click', sendRequest);

    // LOAD DASHBOARD BY DEFAULT
    switchTab('dashboard');
  </script>
</body>
</html>`))
}

// OpsEndpointCatalog returns a curated endpoint list used by the Ops Console.
func OpsEndpointCatalog(w http.ResponseWriter, r *http.Request) {
	endpoints := []opsEndpoint{
		{Method: "GET", Path: "/health", Scope: "public", Notes: "Liveness check"},
		{Method: "GET", Path: "/health/deep", Scope: "public", Notes: "Dependencies + metrics"},
		{Method: "GET", Path: "/readiness", Scope: "public", Notes: "Readiness probe"},
		{Method: "GET", Path: "/metrics", Scope: "public", Notes: "Prometheus-style metrics"},
		{Method: "GET", Path: "/pricing/tiers", Scope: "public", Notes: "Pricing catalog"},
		{Method: "GET", Path: "/discover/trending", Scope: "public", Notes: "Trending discovery"},
		{Method: "GET", Path: "/search?q=...", Scope: "public", Notes: "Global search"},
		{Method: "GET", Path: "/projects", Scope: "auth", Notes: "List user projects"},
		{Method: "POST", Path: "/projects", Scope: "auth", Notes: "Create project"},
		{Method: "GET", Path: "/projects/{id}", Scope: "auth", Notes: "Get one project"},
		{Method: "GET", Path: "/offline/storage", Scope: "auth", Notes: "Offline quota/usage"},
		{Method: "GET", Path: "/user/stats", Scope: "auth", Notes: "User analytics summary"},
		{Method: "POST", Path: "/checkout/session", Scope: "auth", Notes: "Create checkout session"},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(endpoints)
}
