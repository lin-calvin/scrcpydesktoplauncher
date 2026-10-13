import './style.css';

import '@material/web/button/filled-button.js';
import '@material/web/button/outlined-button.js';
import '@material/web/button/text-button.js';
import '@material/web/textfield/outlined-text-field.js';
import '@material/web/select/outlined-select.js';
import '@material/web/select/select-option.js';
import '@material/web/switch/switch.js';
import '@material/web/checkbox/checkbox.js';

const $ = (sel, root = document) => root.querySelector(sel);

const backend = () => (window.go && window.go.main && window.go.main.App) || null;
async function api(name, ...args) {
  const b = backend();
  if (!b || typeof b[name] !== 'function') throw new Error('backend not available');
  return b[name](...args);
}

function defaults() {
  return {
    folder_name: 'Android', scope: 'launchable', include: [], exclude: [],
    prefer_transport: 'auto', audio: true, device: 'auto', settings_entry: true,
    scrcpy: { size: '', dpi: '', flex_display: true, no_vd_decorations: false, keep_content_on_close: false },
  };
}

let cfg = defaults();
let apps = [];
let selected = new Map();

// ---------- markup ----------
$('#app').innerHTML = `
  <div class="appbar">
    <h1>Android Launcher</h1>
    <div class="status"><span class="dot" id="dot"></span><span id="status-text">checking…</span></div>
  </div>
  <main>
    <div class="pane-left">
      <div class="section-title">Launcher</div>
      <md-outlined-text-field id="folder" label="Folder name"></md-outlined-text-field>
      <div class="row"><label>Settings entry in folder</label><md-switch id="settings-entry"></md-switch></div>

      <div class="section-title">Device</div>
      <md-outlined-select id="transport" label="Preferred transport">
        <md-select-option value="auto"><div slot="headline">Automatic</div></md-select-option>
        <md-select-option value="usb"><div slot="headline">USB</div></md-select-option>
        <md-select-option value="tcp"><div slot="headline">TCP/IP</div></md-select-option>
      </md-outlined-select>
      <div class="row"><label>Forward audio to PC</label><md-switch id="audio"></md-switch></div>

      <div class="section-title">scrcpy window</div>
      <div class="two">
        <md-outlined-text-field id="size" label="Size WxH"></md-outlined-text-field>
        <md-outlined-text-field id="dpi" label="DPI"></md-outlined-text-field>
      </div>
      <div class="row"><label>Flex display (fit window)</label><md-switch id="flex"></md-switch></div>
      <div class="row"><label>Hide system decorations</label><md-switch id="decor"></md-switch></div>
      <div class="row"><label>Keep app when closed</label><md-switch id="keep"></md-switch></div>

      <div class="section-title">Actions</div>
      <md-filled-button id="sync">Save &amp; sync now</md-filled-button>
      <md-outlined-button id="save">Save only</md-outlined-button>
      <md-text-button id="clean">Remove all entries</md-text-button>
    </div>

    <div class="pane-right">
      <div class="toolbar">
        <md-outlined-text-field id="search" label="Search apps" style="width:280px"></md-outlined-text-field>
        <md-outlined-select id="scope" label="Scope" style="width:180px">
          <md-select-option value="launchable"><div slot="headline">All launchable</div></md-select-option>
          <md-select-option value="third_party"><div slot="headline">Third-party only</div></md-select-option>
        </md-outlined-select>
        <div class="spacer"></div>
        <span class="count" id="count"></span>
        <md-text-button id="all">All</md-text-button>
        <md-text-button id="none">None</md-text-button>
      </div>
      <div class="applist" id="list"><div class="empty">Loading apps…</div></div>
      <div class="actions">
        <md-filled-button id="sync2">Save &amp; sync now</md-filled-button>
      </div>
    </div>
  </main>
  <div id="toast" style="position:fixed;left:50%;bottom:20px;transform:translateX(-50%);
       background:var(--md-sys-color-primary);color:var(--md-sys-color-on-primary);
       padding:10px 18px;border-radius:10px;font-size:13px;opacity:0;transition:opacity .2s;pointer-events:none"></div>
`;

// ---------- helpers ----------
let toastTimer;
function toast(msg, err) {
  const t = $('#toast');
  t.textContent = msg;
  t.style.background = err ? 'var(--md-sys-color-error)' : 'var(--md-sys-color-primary)';
  t.style.color = err ? 'var(--md-sys-color-on-error-container)' : 'var(--md-sys-color-on-primary)';
  t.style.opacity = '1';
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (t.style.opacity = '0'), 3200);
}

// ---------- config <-> form ----------
function fillForm(c) {
  $('#folder').value = c.folder_name || 'Android';
  $('#settings-entry').selected = !!c.settings_entry;
  $('#transport').value = c.prefer_transport || 'auto';
  $('#audio').selected = !!c.audio;
  $('#size').value = c.scrcpy?.size || '';
  $('#dpi').value = c.scrcpy?.dpi || '';
  $('#flex').selected = !!c.scrcpy?.flex_display;
  $('#decor').selected = !!c.scrcpy?.no_vd_decorations;
  $('#keep').selected = !!c.scrcpy?.keep_content_on_close;
  $('#scope').value = c.scope || 'launchable';
}

function base(scope, a) {
  return scope === 'launchable' ? true : !a.system;
}

function collect() {
  const scope = $('#scope').value || 'launchable';
  const include = [];
  const exclude = [];
  for (const a of apps) {
    const sel = selected.get(a.package) ?? base(scope, a);
    if (sel && !base(scope, a)) include.push(a.package);
    if (!sel && base(scope, a)) exclude.push(a.package);
  }
  return {
    folder_name: $('#folder').value || 'Android',
    scope,
    include,
    exclude,
    prefer_transport: $('#transport').value || 'auto',
    audio: $('#audio').selected,
    device: 'auto',
    settings_entry: $('#settings-entry').selected,
    scrcpy: {
      size: ($('#size').value || '').trim(),
      dpi: ($('#dpi').value || '').trim(),
      flex_display: $('#flex').selected,
      no_vd_decorations: $('#decor').selected,
      keep_content_on_close: $('#keep').selected,
    },
  };
}

// ---------- rendering the app list ----------
const observer = new IntersectionObserver((entries) => {
  for (const e of entries) {
    if (!e.isIntersecting) continue;
    const el = e.target;
    observer.unobserve(el);
    const pkg = el.dataset.pkg;
    api('IconDataURL', pkg).then((url) => {
      if (!url) return;
      let img = el.querySelector('img');
      if (!img) {
        img = document.createElement('img');
        const ph = el.querySelector('.ph');
        if (ph) ph.replaceWith(img);
        else el.prepend(img);
      }
      img.src = url;
    }).catch(() => {});
  }
}, { root: $('#list'), rootMargin: '200px' });

function renderList() {
  const term = ($('#search').value || '').toLowerCase();
  const scope = $('#scope').value || 'launchable';
  const list = $('#list');
  const shown = apps.filter(
    (a) => !term || a.label.toLowerCase().includes(term) || a.package.toLowerCase().includes(term)
  );
  const selectedCount = apps.filter((a) => selected.get(a.package) ?? base(scope, a)).length;
  $('#count').textContent = `${shown.length}/${apps.length} shown · ${selectedCount} selected`;

  if (!shown.length) {
    list.innerHTML = `<div class="empty">${apps.length ? 'No match' : 'No device / no apps'}</div>`;
    return;
  }
  const frag = document.createDocumentFragment();
  for (const a of shown) {
    const row = document.createElement('div');
    row.className = 'app-item';
    row.dataset.pkg = a.package;
    const checked = selected.get(a.package) ?? base(scope, a);
    row.innerHTML = `
      <md-checkbox ${checked ? 'checked' : ''} touch-target="none"></md-checkbox>
      <div class="ph"></div>
      <div class="app-meta">
        <span class="app-name">${escapeHtml(a.label)}${a.system ? '<span class="badge">system</span>' : ''}</span>
        <span class="app-pkg">${escapeHtml(a.package)}</span>
      </div>`;
    row.addEventListener('click', (ev) => {
      if (ev.target.tagName === 'MD-CHECKBOX') return;
      const box = row.querySelector('md-checkbox');
      box.checked = !box.checked;
      selected.set(a.package, box.checked);
      renderList();
    });
    row.querySelector('md-checkbox').addEventListener('change', (ev) => {
      selected.set(a.package, ev.target.checked);
      renderList();
    });
    frag.appendChild(row);
    observer.observe(row);
  }
  list.replaceChildren(frag);
}

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));
}

// ---------- status / load ----------
async function refreshStatus() {
  try {
    const st = await api('DeviceStatus');
    $('#dot').classList.toggle('on', st.connected);
    $('#status-text').textContent = st.connected
      ? `connected: ${st.devices.join(', ')}`
      : 'no device';
    return st;
  } catch {
    $('#status-text').textContent = 'backend offline';
    return { connected: false, devices: [] };
  }
}

async function loadApps() {
  $('#list').innerHTML = '<div class="empty">Loading apps…</div>';
  try {
    apps = await api('ListApps');
    selected = new Map(apps.map((a) => [a.package, a.selected]));
    renderList();
  } catch (e) {
    apps = [];
    $('#list').innerHTML = `<div class="empty">${escapeHtml(String(e))}</div>`;
  }
}

async function save() {
  cfg = collect();
  await api('SaveConfig', cfg);
}

async function sync() {
  try {
    await save();
    const res = await api('ReconcileNow');
    toast(`Synced ${res.total} apps (added ${res.added.length}, removed ${res.removed.length})`);
    await refreshStatus();
  } catch (e) {
    toast(String(e), true);
  }
}

// ---------- events ----------
$('#search').addEventListener('input', renderList);
$('#scope').addEventListener('change', () => {
  for (const a of apps) selected.set(a.package, base($('#scope').value, a));
  renderList();
});
$('#all').addEventListener('click', () => { for (const a of apps) selected.set(a.package, true); renderList(); });
$('#none').addEventListener('click', () => { for (const a of apps) selected.set(a.package, false); renderList(); });
$('#save').addEventListener('click', async () => { try { await save(); toast('Saved'); } catch (e) { toast(String(e), true); } });
$('#sync').addEventListener('click', sync);
$('#sync2').addEventListener('click', sync);
$('#clean').addEventListener('click', async () => {
  try { await api('CleanAll'); toast('Removed all entries'); await refreshStatus(); }
  catch (e) { toast(String(e), true); }
});

// ---------- boot ----------
(async function () {
  try { cfg = await api('GetConfig'); } catch { cfg = defaults(); }
  fillForm(cfg);
  await refreshStatus();
  await loadApps();
})();
