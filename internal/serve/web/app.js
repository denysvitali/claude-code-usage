/* llm usage — quota monitor
 *
 * Answers one question: which limit runs out first, and when does it come back.
 * No build step, no CDN — the binary embeds these three files and serves them
 * locally, so the page has to work with nothing but the platform.
 */
(() => {
  'use strict';

  // Usage endpoints are rate-limited too — poll gently, pause when the window
  // is hidden, and let the refresh button cover the impatient case.
  const REFRESH_MS = 60_000;
  const WARNING_AT = 75;
  const CRITICAL_AT = 90;
  const PACE_MARGIN = 8;   // points of slack before "outrunning the refill"

  const NAMES = {
    claude: 'Claude', codex: 'Codex', grok: 'Grok',
    kimi: 'Kimi', minimax: 'MiniMax', zai: 'Z.AI',
  };

  const el = (id) => document.getElementById(id);
  const root = document.documentElement;
  const dom = {
    brandMark: el('brandMark'), freshness: el('freshness'), filters: el('filters'),
    banner: el('banner'), hero: el('hero'), grid: el('grid'), blank: el('blank'),
    favicon: el('favicon'), sortBtn: el('sortBtn'), sortLabel: el('sortLabel'),
    densityBtn: el('densityBtn'), themeBtn: el('themeBtn'), refreshBtn: el('refreshBtn'),
  };
  const tplCard = el('tpl-card').content;
  const tplWindow = el('tpl-window').content;

  // Colour + shape, set together — see the .dot rules in app.css.
  function markProvider(node, id) {
    node.dataset.p = id;
    node.style.setProperty('--accent', `var(--p-${id}, var(--ink-3))`);
  }

  const state = {
    reports: null,
    fetchedAt: null,
    loading: false,
    failure: null,
    timer: null,
    cards: new Map(),
    hero: null,
    favicon: '',
  };

  const prefs = loadPrefs();

  /* ── Preferences ──────────────────────────────────────────────────── */

  function loadPrefs() {
    let result = { theme: 'auto', density: 'auto', sort: 'urgency', hidden: [] };
    try {
      const saved = JSON.parse(localStorage.getItem('llm-usage.prefs') || '{}');
      result = { ...result, ...saved, hidden: Array.isArray(saved.hidden) ? saved.hidden : [] };
    } catch { /* private mode */ }

    // URL pins win for this load; they're only persisted if the user then
    // touches the control themselves.
    const query = new URLSearchParams(location.search);
    if (/^(auto|dark|light)$/.test(query.get('theme'))) result.theme = query.get('theme');
    if (/^(auto|dense)$/.test(query.get('density'))) result.density = query.get('density');
    if (/^(urgency|name)$/.test(query.get('sort'))) result.sort = query.get('sort');
    return result;
  }

  function savePrefs() {
    try { localStorage.setItem('llm-usage.prefs', JSON.stringify(prefs)); } catch { /* private mode */ }
  }

  /* ── Formatting ───────────────────────────────────────────────────── */

  function providerName(id) {
    return NAMES[id] || (id || '').replace(/^./, (c) => c.toUpperCase());
  }

  function statusOf(pct) {
    if (pct >= CRITICAL_AT) return 'critical';
    if (pct >= WARNING_AT) return 'warning';
    return 'good';
  }

  function fmtPct(v) {
    const n = Number(v) || 0;
    return (n >= 99.95 || Number.isInteger(n) ? Math.round(n) : n.toFixed(1)) + '%';
  }

  function fmtNum(v) {
    const n = Number(v) || 0;
    if (Math.abs(n) >= 1e9) return (n / 1e9).toFixed(1).replace(/\.0$/, '') + 'B';
    if (Math.abs(n) >= 1e6) return (n / 1e6).toFixed(1).replace(/\.0$/, '') + 'M';
    if (Math.abs(n) >= 10_000) return (n / 1e3).toFixed(1).replace(/\.0$/, '') + 'k';
    return String(Math.round(n * 10) / 10);
  }

  // Compact, always two units at most: 6d 4h · 4h 22m · 16m 04s · 42s
  function fmtLeft(ms) {
    if (ms <= 0) return 'now';
    const s = Math.floor(ms / 1000);
    const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600);
    const m = Math.floor((s % 3600) / 60), sec = s % 60;
    if (d) return `${d}d ${h}h`;
    if (h) return `${h}h ${m}m`;
    if (m >= 10) return `${m}m`;
    if (m) return `${m}m ${String(sec).padStart(2, '0')}s`;
    return `${sec}s`;
  }

  function fmtAgo(ms) {
    const s = Math.max(0, Math.round(ms / 1000));
    if (s < 5) return 'just now';
    if (s < 60) return `${s}s ago`;
    const m = Math.floor(s / 60);
    if (m < 60) return `${m}m ago`;
    return `${Math.floor(m / 60)}h ago`;
  }

  /* ── The pace signal ──────────────────────────────────────────────── */

  // Window labels state their own duration ("5-Hour", "300-Minute Rate Limit"),
  // and resets_at gives the end — so we know how far through the window we are.
  const UNIT_MS = {
    second: 1e3, sec: 1e3, minute: 6e4, min: 6e4, hour: 36e5, hr: 36e5,
    day: 864e5, week: 6048e5, month: 2592e6,
  };

  function windowSpan(label) {
    const m = /(\d+)\s*[-\s]?\s*(second|sec|minute|min|hour|hr|day|week|month)/i.exec(label || '');
    if (!m) return null;
    const unit = UNIT_MS[m[2].toLowerCase()];
    return unit ? Number(m[1]) * unit : null;
  }

  // Fraction of the window already elapsed, 0–100, or null when unknowable.
  function paceOf(win, now) {
    const span = windowSpan(win.label);
    if (!span || !win.resets_at) return null;
    const end = Date.parse(win.resets_at);
    if (!Number.isFinite(end)) return null;
    const fraction = (span - (end - now)) / span;
    if (fraction < 0 || fraction > 1) return null;
    return fraction * 100;
  }

  function isOverPace(win, pace) {
    return pace !== null && win.utilization >= 10 && win.utilization > pace + PACE_MARGIN;
  }

  /* ── Data ─────────────────────────────────────────────────────────── */

  function keyOf(report) {
    return `${report.provider}/${report.extra?.account || 'default'}`;
  }

  function peakOf(report) {
    const wins = report.windows || [];
    return wins.reduce((max, w) => Math.max(max, Number(w.utilization) || 0), 0);
  }

  function soonestReset(report) {
    let soonest = Infinity;
    for (const w of report.windows || []) {
      const t = w.resets_at ? Date.parse(w.resets_at) : NaN;
      if (Number.isFinite(t)) soonest = Math.min(soonest, t);
    }
    return soonest;
  }

  function visibleReports() {
    const all = state.reports || [];
    const list = all.filter((r) => !prefs.hidden.includes(r.provider));

    return list.sort((a, b) => {
      if (prefs.sort === 'name') {
        return keyOf(a).localeCompare(keyOf(b));
      }
      if (Boolean(a.error) !== Boolean(b.error)) return a.error ? 1 : -1;
      const diff = peakOf(b) - peakOf(a);
      return diff !== 0 ? diff : soonestReset(a) - soonestReset(b);
    });
  }

  // The single tightest constraint across everything currently shown.
  function tightest(reports) {
    let best = null;
    for (const report of reports) {
      if (report.error) continue;
      for (const win of report.windows || []) {
        const pct = Number(win.utilization) || 0;
        if (!best || pct > best.pct) best = { report, win, pct };
      }
    }
    return best;
  }

  async function refresh() {
    if (state.loading) return;
    state.loading = true;
    root.dataset.loading = '1';
    paintFreshness();

    try {
      const res = await fetch('api/v1/usage', { headers: { Accept: 'application/json' } });
      if (!res.ok) throw new Error(`server returned HTTP ${res.status}`);
      const body = await res.json();
      state.reports = Array.isArray(body.providers) ? body.providers : [];
      state.fetchedAt = Date.now();
      state.failure = null;
    } catch (err) {
      state.failure = err.message || String(err);
    } finally {
      state.loading = false;
      delete root.dataset.loading;
      render();
    }
  }

  /* ── Rendering ────────────────────────────────────────────────────── */

  function render() {
    const reports = visibleReports();

    paintBanner();
    paintFilters();
    paintHero(reports);
    paintCards(reports);
    paintFreshness();
    paintGlobal(reports);
    tick();

    if (state.reports === null || reports.length) {
      dom.blank.hidden = true;
    } else {
      dom.blank.hidden = false;
      dom.blank.innerHTML = state.reports.length
        ? '<p>Every provider is filtered out.</p><p>Pick one above to bring it back.</p>'
        : '<p>No providers configured.</p>' +
          '<p>Run <code>llm-usage setup add</code> to connect an account, then refresh.</p>';
    }
  }

  function paintBanner() {
    if (!state.failure) { dom.banner.hidden = true; return; }
    dom.banner.hidden = false;
    dom.banner.textContent = '';
    const label = document.createElement('b');
    label.textContent = state.reports ? 'Showing the last good read.' : "Can't reach the server.";
    const detail = document.createElement('span');
    detail.textContent = state.failure;
    dom.banner.append(label, detail);
  }

  function paintFilters() {
    const ids = [...new Set((state.reports || []).map((r) => r.provider))];
    if (ids.length < 2) { dom.filters.hidden = true; return; }
    dom.filters.hidden = false;

    const wanted = ids.join('|');
    if (dom.filters.dataset.ids !== wanted) {
      dom.filters.dataset.ids = wanted;
      dom.filters.textContent = '';
      for (const id of ids) {
        const pill = document.createElement('button');
        pill.type = 'button';
        pill.className = 'pill';
        pill.dataset.provider = id;
        const dot = document.createElement('span');
        dot.className = 'dot';
        markProvider(dot, id);
        pill.append(dot, document.createTextNode(providerName(id)));
        pill.addEventListener('click', () => toggleProvider(id));
        dom.filters.append(pill);
      }
    }
    for (const pill of dom.filters.children) {
      pill.setAttribute('aria-pressed', String(!prefs.hidden.includes(pill.dataset.provider)));
    }
    markScrollable();
  }

  // The edge fade is an affordance, so it only appears when there is in fact
  // more to scroll to.
  function markScrollable() {
    const overflowing = dom.filters.scrollWidth > dom.filters.clientWidth + 2;
    dom.filters.dataset.scrollable = overflowing ? '1' : '0';
  }

  function toggleProvider(id) {
    const at = prefs.hidden.indexOf(id);
    if (at > -1) prefs.hidden.splice(at, 1);
    else prefs.hidden.push(id);
    savePrefs();
    render();
  }

  function paintHero(reports) {
    const top = tightest(reports);
    if (!top) { dom.hero.hidden = true; state.hero = null; return; }
    dom.hero.hidden = false;

    if (!state.hero) {
      dom.hero.innerHTML =
        '<p class="hero-eyebrow"><span class="dot"></span><span class="hero-scope"></span></p>' +
        '<div class="hero-line">' +
          '<span class="hero-what"></span><span class="hero-pct"></span>' +
          '<span class="hero-when"></span>' +
        '</div>' +
        '<div class="meter"><div class="meter-fill"></div><div class="meter-pace" hidden></div></div>' +
        '<p class="hero-note"><span class="tickmark"></span><span class="hero-legend"></span></p>';
      state.hero = {
        dot: dom.hero.querySelector('.dot'),
        scope: dom.hero.querySelector('.hero-scope'),
        what: dom.hero.querySelector('.hero-what'),
        pct: dom.hero.querySelector('.hero-pct'),
        when: dom.hero.querySelector('.hero-when'),
        meter: dom.hero.querySelector('.meter'),
        legend: dom.hero.querySelector('.hero-legend'),
        note: dom.hero.querySelector('.hero-note'),
      };
    }

    const h = state.hero;
    markProvider(h.dot, top.report.provider);
    h.scope.textContent = 'Tightest limit';
    h.what.textContent = `${providerName(top.report.provider)} · ${top.win.label}`;
    h.pct.textContent = fmtPct(top.pct);
    paintMeter(h.meter, top.win, Date.now());
    h.meter.dataset.win = JSON.stringify({ label: top.win.label, resets_at: top.win.resets_at, utilization: top.pct });
  }

  function paintCards(reports) {
    for (const ghost of dom.grid.querySelectorAll('.skel')) ghost.remove();
    const seen = new Set();
    let previous = null;

    for (const report of reports) {
      const key = keyOf(report);
      seen.add(key);
      let card = state.cards.get(key);
      if (!card) {
        card = buildCard();
        state.cards.set(key, card);
      }
      updateCard(card, report);
      // Keep DOM order in sync with sort order without rebuilding.
      const shouldFollow = previous ? previous.nextElementSibling : dom.grid.firstElementChild;
      if (shouldFollow !== card.root) dom.grid.insertBefore(card.root, shouldFollow);
      previous = card.root;
    }

    for (const [key, card] of state.cards) {
      if (!seen.has(key)) { card.root.remove(); state.cards.delete(key); }
    }
  }

  function buildCard() {
    const root = tplCard.firstElementChild.cloneNode(true);
    return {
      root,
      dot: root.querySelector('.dot'),
      name: root.querySelector('.card-name'),
      account: root.querySelector('.card-account'),
      chip: root.querySelector('.chip'),
      body: root.querySelector('.card-body'),
      wins: new Map(),
      signature: '',
    };
  }

  function updateCard(card, report) {
    const now = Date.now();
    const account = report.extra?.account;
    markProvider(card.dot, report.provider);
    card.name.textContent = providerName(report.provider);
    card.account.textContent = account && account !== 'default' ? account : '';
    card.root.setAttribute('aria-label',
      providerName(report.provider) + (account && account !== 'default' ? ` (${account})` : ''));

    const windows = report.error ? [] : (report.windows || []);
    const peak = peakOf(report);

    if (report.error) {
      card.chip.dataset.status = 'down';
      card.chip.textContent = 'No data';
      card.root.dataset.state = 'down';
    } else if (!windows.length) {
      // No windows is not the same as zero usage — don't claim it's healthy.
      card.chip.dataset.status = 'down';
      card.chip.textContent = '—';
      card.root.dataset.state = 'ok';
    } else {
      card.chip.dataset.status = statusOf(peak);
      card.chip.textContent = fmtPct(peak);
      card.chip.title = 'Highest usage across this account';
      card.root.dataset.state = 'ok';
    }

    // Rebuild the body only when its shape changes; otherwise update in place so
    // meters animate and countdowns don't blink.
    const signature = [
      report.error ? 'e:' + report.error.message : '',
      windows.map((w) => w.label).join('|'),
      report.extra?.extra_usage ? 'x' : '',
      report.extra?.subscription ? 'sub:' + Object.keys(report.extra.subscription).join(',') : '',
    ].join('#');

    if (card.signature !== signature) {
      card.signature = signature;
      card.body.textContent = '';
      card.wins.clear();
      if (report.error) card.body.append(errorNote(report));
      for (const win of windows) card.body.append(buildWindow(card, win));
      const extras = extraNotes(report);
      // "No limits" plus a note explaining why says the same thing twice.
      if (!report.error && !windows.length && !extras.length) {
        card.body.append(quietNote('No limits reported for this account.'));
      }
      for (const node of extras) card.body.append(node);
    }

    for (const win of windows) {
      const parts = card.wins.get(win.label);
      if (parts) updateWindow(parts, win, now);
    }
    updateExtras(card, report);
  }

  function buildWindow(card, win) {
    const root = tplWindow.firstElementChild.cloneNode(true);
    const parts = {
      root,
      label: root.querySelector('.win-label'),
      pct: root.querySelector('.win-pct'),
      inline: root.querySelector('.win-inline'),
      meter: root.querySelector('.meter'),
      amounts: root.querySelector('.win-amounts'),
      reset: root.querySelector('.win-reset'),
    };
    parts.label.textContent = win.label;
    card.wins.set(win.label, parts);
    return root;
  }

  function updateWindow(parts, win, now) {
    const pct = Number(win.utilization) || 0;
    parts.pct.textContent = fmtPct(pct);
    parts.data = win;
    paintMeter(parts.meter, win, now);

    if (win.used != null && win.limit != null) {
      parts.amounts.textContent = `${fmtNum(win.used)} of ${fmtNum(win.limit)}`;
    } else if (win.remaining != null) {
      parts.amounts.textContent = `${fmtNum(win.remaining)} left`;
    } else {
      parts.amounts.textContent = '';
    }
    paintWindowTime(parts, now);
  }

  // Split out so the 1s tick can update countdowns without touching anything else.
  function paintWindowTime(parts, now) {
    const win = parts.data;
    if (!win) return;
    const pace = paceOf(win, now);
    const over = isOverPace(win, pace);

    if (!win.resets_at) {
      parts.reset.textContent = 'no reset';
      parts.inline.textContent = '';
      parts.reset.removeAttribute('data-over');
      parts.inline.removeAttribute('data-over');
      return;
    }

    const left = fmtLeft(Date.parse(win.resets_at) - now);
    parts.reset.textContent = over ? `▲ resets in ${left}` : `resets in ${left}`;
    parts.inline.textContent = over ? `▲ ${left}` : left;
    parts.reset.dataset.over = over ? '1' : '0';
    parts.inline.dataset.over = over ? '1' : '0';
    const hint = over
      ? `Spending faster than this window refills. Resets ${new Date(win.resets_at).toLocaleString()}.`
      : `Resets ${new Date(win.resets_at).toLocaleString()}.`;
    parts.reset.title = hint;
    parts.inline.title = hint;
  }

  function paintMeter(meter, win, now) {
    const pct = Math.max(0, Math.min(100, Number(win.utilization) || 0));
    const fill = meter.querySelector('.meter-fill');
    const tick = meter.querySelector('.meter-pace');
    const status = statusOf(pct);

    meter.dataset.status = status;
    meter.setAttribute('role', 'meter');
    meter.setAttribute('aria-valuemin', '0');
    meter.setAttribute('aria-valuemax', '100');
    meter.setAttribute('aria-valuenow', String(Math.round(pct)));
    fill.style.width = pct > 0 ? `max(${pct}%, 7px)` : '0';

    const pace = paceOf(win, now);
    if (pace === null) {
      tick.hidden = true;
    } else {
      const over = isOverPace(win, pace);
      tick.hidden = false;
      tick.style.left = `${pace}%`;
      tick.dataset.over = over ? '1' : '0';
      tick.title = `${Math.round(pace)}% of this window has elapsed`;
    }
    meter.setAttribute('aria-label',
      `${win.label}: ${fmtPct(pct)} used` + (pace === null ? '' : `, ${Math.round(pace)}% of the window elapsed`));
  }

  /* ── Notes ────────────────────────────────────────────────────────── */

  // Upstream failures arrive as "Provider: request failed with status 429: {json}".
  // Lift the human sentence out of the payload and keep the status code.
  function tidyError(raw) {
    let text = (raw || 'Usage could not be read.').replace(/^[^:]+:\s*/, '');
    let detail = '';
    const brace = text.indexOf('{');
    if (brace > -1) {
      try {
        const parsed = JSON.parse(text.slice(brace));
        detail = parsed?.error?.message || parsed?.message || '';
      } catch { /* not JSON after all */ }
      text = text.slice(0, brace).replace(/[\s:]+$/, '');
    }
    if (detail) {
      const code = /\b(\d{3})\b/.exec(text);
      text = code ? `${detail} (HTTP ${code[1]})` : detail;
    }
    return text.replace(/^./, (c) => c.toUpperCase());
  }

  function errorNote(report) {
    const box = document.createElement('div');
    box.className = 'note note-down';
    const message = report.error?.message || '';
    box.textContent = tidyError(message);
    if (/401|403|unauthor|token|credential|cookie|log in/i.test(message)) {
      const hint = document.createElement('p');
      hint.className = 'note-hint';
      const command = document.createElement('code');
      command.textContent = `llm-usage setup add ${report.provider}`;
      hint.append('Credentials look stale — run ', command);
      box.append(hint);
    }
    return box;
  }

  function quietNote(text) {
    const box = document.createElement('div');
    box.className = 'note';
    box.textContent = text;
    return box;
  }

  function extraNotes(report) {
    const notes = [];
    const extra = report.extra || {};

    if (extra.extra_usage) {
      const box = document.createElement('div');
      box.className = 'note';
      box.dataset.role = 'credits';
      box.innerHTML =
        '<span class="note-title">Extra credits</span>' +
        '<div class="meter"><div class="meter-fill"></div><div class="meter-pace" hidden></div></div>' +
        '<div class="kv" style="margin-top:6px"><span class="kv-k"></span><span class="kv-v"></span></div>';
      notes.push(box);
    }

    const sub = extra.subscription;
    if (sub && (sub.plan || sub.features || sub.subscribed !== undefined)) {
      const box = document.createElement('div');
      box.className = 'note';
      box.dataset.role = 'subscription';
      notes.push(box);
    } else if (sub && typeof sub.status === 'string') {
      notes.push(quietNote(sub.status.replace(/^./, (c) => c.toUpperCase())));
    }
    return notes;
  }

  function updateExtras(card, report) {
    const extra = report.extra || {};

    const credits = card.body.querySelector('[data-role="credits"]');
    if (credits && extra.extra_usage) {
      const used = (extra.extra_usage.used_credits || 0) / 100;
      const limit = (extra.extra_usage.monthly_limit || 0) / 100;
      paintMeter(credits.querySelector('.meter'), { label: 'Extra credits', utilization: extra.extra_usage.utilization || 0 }, Date.now());
      credits.querySelector('.kv-k').textContent = 'Spent this month';
      credits.querySelector('.kv-v').textContent = `$${used.toFixed(2)} of $${limit.toFixed(2)}`;
    }

    const subBox = card.body.querySelector('[data-role="subscription"]');
    if (subBox && extra.subscription) {
      const sub = extra.subscription;
      subBox.textContent = '';
      const title = document.createElement('span');
      title.className = 'note-title';
      title.textContent = 'Plan';
      subBox.append(title);

      if (sub.plan) {
        subBox.append(kv(sub.plan.title || 'Subscription', [sub.plan.level, sub.plan.status].filter(Boolean).join(' · ')));
      } else {
        subBox.append(kv('Subscription', sub.subscribed ? 'Active' : 'None'));
      }
      if (sub.expires_at) {
        const ms = Date.parse(sub.expires_at) - Date.now();
        const when = new Date(sub.expires_at).toLocaleDateString();
        subBox.append(kv(ms < 0 ? 'Expired' : 'Renews', ms < 0 ? when : `${when} · ${fmtLeft(ms)}`));
      }
      for (const feature of sub.features || []) {
        subBox.append(kv(feature.feature, `${feature.left} of ${feature.total} left`));
      }
    }
  }

  function kv(label, value) {
    const row = document.createElement('div');
    row.className = 'kv';
    const k = document.createElement('span');
    k.textContent = label;
    const v = document.createElement('span');
    v.className = 'kv-v';
    v.textContent = value;
    row.append(k, v);
    return row;
  }

  /* ── Chrome ───────────────────────────────────────────────────────── */

  function paintFreshness() {
    if (state.loading && !state.reports) { dom.freshness.textContent = 'Reading limits'; return; }
    if (!state.fetchedAt) { dom.freshness.textContent = state.failure ? 'No data' : 'Reading limits'; return; }
    dom.freshness.textContent = `Updated ${fmtAgo(Date.now() - state.fetchedAt)}`;
    dom.freshness.dataset.stale = state.failure ? '1' : '0';
  }

  function paintGlobal(reports) {
    let peak = 0;
    let anyHealthy = false;
    for (const report of reports) {
      if (report.error) continue;
      anyHealthy = true;
      peak = Math.max(peak, peakOf(report));
    }
    const status = anyHealthy ? statusOf(peak) : 'down';
    dom.brandMark.dataset.status = status;
    document.title = anyHealthy ? `${Math.round(peak)}% · llm usage` : 'llm usage';
    paintFavicon(anyHealthy ? peak : 0, status);
  }

  // A tiny window may show nothing but the tab — so the tab carries the number.
  const FAVICON_INK = { good: '#0ca30c', warning: '#fab219', critical: '#d03b3b', down: '#7f8a99' };

  function paintFavicon(pct, status) {
    const stamp = `${Math.round(pct)}/${status}`;
    if (state.favicon === stamp) return;
    state.favicon = stamp;

    const size = 32;
    const canvas = document.createElement('canvas');
    canvas.width = canvas.height = size;
    const ctx = canvas.getContext('2d');
    if (!ctx) return;
    const radius = 12;
    ctx.lineWidth = 5;
    ctx.lineCap = 'round';
    ctx.strokeStyle = 'rgba(140,150,165,0.35)';
    ctx.beginPath();
    ctx.arc(size / 2, size / 2, radius, 0, Math.PI * 2);
    ctx.stroke();
    if (pct > 0) {
      ctx.strokeStyle = FAVICON_INK[status] || FAVICON_INK.down;
      ctx.beginPath();
      ctx.arc(size / 2, size / 2, radius, -Math.PI / 2, -Math.PI / 2 + (Math.PI * 2 * Math.min(pct, 100)) / 100);
      ctx.stroke();
    }
    dom.favicon.href = canvas.toDataURL('image/png');
  }

  function tick() {
    const now = Date.now();
    for (const card of state.cards.values()) {
      for (const parts of card.wins.values()) paintWindowTime(parts, now);
    }
    if (state.hero?.meter.dataset.win) {
      const win = JSON.parse(state.hero.meter.dataset.win);
      const pace = paceOf(win, now);
      const over = isOverPace(win, pace);
      const left = win.resets_at ? fmtLeft(Date.parse(win.resets_at) - now) : null;

      state.hero.when.textContent = left
        ? `${over ? '▲ ' : ''}resets in ${left}`
        : 'no scheduled reset';
      state.hero.when.style.color = over ? 'var(--serious)' : '';
      state.hero.legend.textContent = pace === null
        ? 'Fill shows how much of the allowance is spent.'
        : over
          ? 'Past the mark — spending faster than this window refills.'
          : 'The mark shows how far into the window you are.';
      paintMeter(state.hero.meter, win, now);
    }
    paintFreshness();
  }

  /* ── Controls ─────────────────────────────────────────────────────── */

  function applyPrefs() {
    root.dataset.theme = prefs.theme;
    root.dataset.density = prefs.density;
    dom.sortLabel.textContent = prefs.sort === 'name' ? 'Name' : 'Urgency';
    dom.sortBtn.setAttribute('aria-label', `Sorted by ${prefs.sort}. Switch sort order`);
    dom.themeBtn.setAttribute('aria-label', `Theme: ${prefs.theme}. Switch theme`);
    dom.densityBtn.setAttribute('aria-label', `Density: ${prefs.density}. Switch density`);
  }

  function cycle(name, values) {
    prefs[name] = values[(values.indexOf(prefs[name]) + 1) % values.length];
    savePrefs();
    applyPrefs();
  }

  dom.refreshBtn.addEventListener('click', refresh);
  dom.sortBtn.addEventListener('click', () => { cycle('sort', ['urgency', 'name']); render(); });
  dom.themeBtn.addEventListener('click', () => cycle('theme', ['auto', 'dark', 'light']));
  dom.densityBtn.addEventListener('click', () => cycle('density', ['auto', 'dense']));

  document.addEventListener('keydown', (event) => {
    if (event.metaKey || event.ctrlKey || event.altKey) return;
    if (/^(input|textarea|select)$/i.test(event.target.tagName)) return;
    const key = event.key.toLowerCase();
    if (key === 'r') { event.preventDefault(); refresh(); }
    else if (key === 't') { event.preventDefault(); cycle('theme', ['auto', 'dark', 'light']); }
    else if (key === 'd') { event.preventDefault(); cycle('density', ['auto', 'dense']); }
    else if (key === 's') { event.preventDefault(); cycle('sort', ['urgency', 'name']); render(); }
  });

  // Don't poll a window nobody is looking at; catch up the moment it returns.
  document.addEventListener('visibilitychange', () => {
    if (document.hidden) {
      clearInterval(state.timer);
      state.timer = null;
    } else {
      if (!state.timer) state.timer = setInterval(refresh, REFRESH_MS);
      if (!state.fetchedAt || Date.now() - state.fetchedAt > 10_000) refresh();
    }
  });

  if (typeof ResizeObserver === 'function') {
    new ResizeObserver(markScrollable).observe(dom.filters);
  }

  dom.grid.innerHTML = '<div class="skel"></div><div class="skel"></div><div class="skel"></div>';
  applyPrefs();
  refresh().then(() => { dom.grid.querySelectorAll('.skel').forEach((s) => s.remove()); });
  state.timer = setInterval(refresh, REFRESH_MS);
  setInterval(tick, 1000);
})();
