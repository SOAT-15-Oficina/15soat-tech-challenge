// ── API Client ──────────────────────────────────────────────────────────────

const api = {
  getToken: () => localStorage.getItem('token'),
  setToken: (t) => localStorage.setItem('token', t),
  clearToken: () => localStorage.removeItem('token'),

  async request(method, path, body) {
    const opts = { method, headers: {} };
    const token = this.getToken();
    if (token) opts.headers['Authorization'] = `Bearer ${token}`;
    if (body) {
      opts.headers['Content-Type'] = 'application/json';
      opts.body = JSON.stringify(body);
    }
    const resp = await fetch(path, opts);
    if (resp.status === 401) {
      this.clearToken();
      window.location.href = '/web/index.html';
      throw new Error('Unauthorized');
    }
    return resp;
  },

  get(path) { return this.request('GET', path); },
  post(path, body) { return this.request('POST', path, body); },
  put(path, body) { return this.request('PUT', path, body); },
  del(path) { return this.request('DELETE', path); },

  async json(resp) {
    if (!resp.ok) {
      const err = await resp.json().catch(() => ({ error: resp.statusText }));
      throw new Error(err.error || resp.statusText);
    }
    return resp.json();
  }
};

// ── Auth helpers ────────────────────────────────────────────────────────────

function requireAuth() {
  if (!api.getToken()) { window.location.href = '/web/index.html'; return false; }
  return true;
}

function parseJwt(token) {
  try {
    const b = token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/');
    return JSON.parse(atob(b));
  } catch { return null; }
}

function getUser() {
  const t = api.getToken();
  return t ? parseJwt(t) : null;
}

function logout() { api.clearToken(); window.location.href = '/web/index.html'; }

// ── Status helpers ──────────────────────────────────────────────────────────

const STATUS_META = {
  RECEBIDA:             { label: 'Recebida',          bg: 'bg-blue-100',    text: 'text-blue-800'    },
  EM_DIAGNOSTICO:       { label: 'Em Diagnostico',    bg: 'bg-yellow-100',  text: 'text-yellow-800'  },
  AGUARDANDO_APROVACAO: { label: 'Aguard. Aprovacao', bg: 'bg-orange-100',  text: 'text-orange-800'  },
  APROVADO:             { label: 'Aprovado',          bg: 'bg-green-100',   text: 'text-green-800'   },
  EM_EXECUCAO:          { label: 'Em Execucao',       bg: 'bg-purple-100',  text: 'text-purple-800'  },
  FINALIZADA:           { label: 'Finalizada',        bg: 'bg-teal-100',    text: 'text-teal-800'    },
  ENTREGUE:             { label: 'Entregue',          bg: 'bg-emerald-100', text: 'text-emerald-800' },
  CANCELADA:            { label: 'Cancelada',         bg: 'bg-red-100',     text: 'text-red-800'     },
};

const STATUS_ORDER = [
  'RECEBIDA','EM_DIAGNOSTICO','AGUARDANDO_APROVACAO','APROVADO','EM_EXECUCAO','FINALIZADA','ENTREGUE'
];

const STATUS_ACTIONS = {
  RECEBIDA:       { next: 'EM_DIAGNOSTICO',       label: 'Iniciar Diagnostico',  cls: 'bg-yellow-500 hover:bg-yellow-600' },
  EM_DIAGNOSTICO: { next: 'AGUARDANDO_APROVACAO', label: 'Enviar Orcamento',     cls: 'bg-orange-500 hover:bg-orange-600' },
  APROVADO:       { next: 'EM_EXECUCAO',          label: 'Iniciar Execucao',     cls: 'bg-purple-500 hover:bg-purple-600' },
  FINALIZADA:     { next: 'ENTREGUE',             label: 'Registrar Entrega',    cls: 'bg-emerald-500 hover:bg-emerald-600' },
};

function statusBadge(status) {
  const m = STATUS_META[status] || { label: status, bg: 'bg-gray-100', text: 'text-gray-800' };
  return `<span class="inline-block px-2 py-1 text-xs font-semibold rounded-full ${m.bg} ${m.text}">${m.label}</span>`;
}

function formatCents(c) { return 'R$ ' + (c / 100).toFixed(2).replace('.', ','); }

function formatDate(d) {
  if (!d) return '';
  return new Date(d).toLocaleString('pt-BR', { dateStyle: 'short', timeStyle: 'short' });
}

// ── Layout ──────────────────────────────────────────────────────────────────

const NAV = [
  { href: '/web/board.html',    label: 'Board'     },
  { href: '/web/clientes.html', label: 'Clientes'  },
  { href: '/web/veiculos.html', label: 'Veiculos'  },
  { href: '/web/servicos.html', label: 'Servicos'  },
  { href: '/web/insumos.html',  label: 'Insumos'   },
  { href: '/web/nova-os.html',  label: 'Nova OS'   },
];

function initLayout() {
  if (!requireAuth()) return;
  const user = getUser();
  const cur = window.location.pathname;

  const sidebar = document.getElementById('sidebar');
  if (sidebar) {
    sidebar.innerHTML = `
      <div class="p-5 border-b border-gray-700">
        <h1 class="text-lg font-bold tracking-wide">Oficina</h1>
        <p class="text-xs text-gray-400 mt-1">Sistema de Gestao</p>
      </div>
      <nav class="flex-1 p-3 space-y-1">
        ${NAV.map(n => `
          <a href="${n.href}" class="block px-3 py-2 rounded text-sm
            ${cur === n.href ? 'bg-gray-700 text-white font-medium' : 'text-gray-300 hover:bg-gray-800 hover:text-white'}">
            ${n.label}
          </a>`).join('')}
      </nav>`;
  }

  const header = document.getElementById('header');
  if (header) {
    header.innerHTML = `
      <div class="text-sm text-gray-500" id="page-title"></div>
      <div class="flex items-center gap-4">
        <span class="text-sm text-gray-600">${user?.user || ''} <span class="text-gray-400">(${user?.role || ''})</span></span>
        <button onclick="logout()" class="text-sm text-red-600 hover:text-red-800 font-medium">Sair</button>
      </div>`;
  }
}

function setPageTitle(t) {
  const el = document.getElementById('page-title');
  if (el) el.textContent = t;
  document.title = 'Oficina - ' + t;
}

// ── Toast ───────────────────────────────────────────────────────────────────

function showToast(msg, type) {
  const el = document.createElement('div');
  el.className = `fixed top-4 right-4 z-50 px-5 py-3 rounded-lg shadow-lg text-white text-sm transition-opacity duration-300
    ${type === 'error' ? 'bg-red-600' : 'bg-green-600'}`;
  el.textContent = msg;
  document.body.appendChild(el);
  setTimeout(() => { el.style.opacity = '0'; setTimeout(() => el.remove(), 300); }, 3000);
}

function showError(msg) { showToast(msg, 'error'); }

// ── Reusable modal ─────────────────────────────────────────────────────────

function openModal(title, bodyHtml) {
  closeModal();
  const overlay = document.createElement('div');
  overlay.id = 'modal-overlay';
  overlay.className = 'fixed inset-0 z-40 bg-black/40 flex items-center justify-center';
  overlay.innerHTML = `
    <div class="bg-white rounded-xl shadow-2xl w-full max-w-lg mx-4 max-h-[90vh] overflow-y-auto">
      <div class="flex items-center justify-between px-6 py-4 border-b">
        <h3 class="text-lg font-semibold">${title}</h3>
        <button onclick="closeModal()" class="text-gray-400 hover:text-gray-600 text-xl">&times;</button>
      </div>
      <div class="p-6">${bodyHtml}</div>
    </div>`;
  overlay.addEventListener('click', e => { if (e.target === overlay) closeModal(); });
  document.body.appendChild(overlay);
}

function closeModal() {
  document.getElementById('modal-overlay')?.remove();
}
