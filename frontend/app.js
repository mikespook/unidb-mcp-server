// API Base URL
const API_BASE = '/api';

// --- Auth ---

async function checkAuth() {
	const res = await fetch('/api/ui/me');
	if (res.status === 401) {
		showLoginScreen();
	} else {
		showMainContent();
	}
}

function showLoginScreen() {
	document.getElementById('login-screen').classList.add('active');
	setTimeout(() => document.getElementById('login-password').focus(), 50);
}

function showMainContent() {
	document.getElementById('login-screen').classList.remove('active');
	document.getElementById('btn-change-password').classList.remove('hidden');
	document.getElementById('btn-logout').classList.remove('hidden');
	loadDSNs();
}

async function login() {
	const password = document.getElementById('login-password').value;
	const errEl = document.getElementById('login-error');
	errEl.textContent = '';

	const res = await fetch('/login', {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ password })
	});

	if (res.ok) {
		document.getElementById('login-password').value = '';
		showMainContent();
	} else {
		const data = await res.json().catch(() => ({}));
		errEl.textContent = data.error || 'Login failed';
	}
}

async function logout() {
	await fetch('/logout', { method: 'POST' });
	document.getElementById('btn-change-password').classList.add('hidden');
	document.getElementById('btn-logout').classList.add('hidden');
	showLoginScreen();
}

function openChangePasswordModal() {
	document.getElementById('cp-current').value = '';
	document.getElementById('cp-new').value = '';
	document.getElementById('cp-confirm').value = '';
	document.getElementById('change-password-modal').classList.add('active');
}

function closeChangePasswordModal() {
	document.getElementById('change-password-modal').classList.remove('active');
}

async function changePassword() {
	const current = document.getElementById('cp-current').value;
	const newPass = document.getElementById('cp-new').value;
	const confirm = document.getElementById('cp-confirm').value;

	if (!current || !newPass) {
		showAlert('Current and new password are required', 'error');
		return;
	}
	if (newPass !== confirm) {
		showAlert('New passwords do not match', 'error');
		return;
	}

	const res = await fetch(`${API_BASE}/ui/password`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ current, new: newPass })
	});

	const data = await res.json().catch(() => ({}));
	if (res.ok && data.success) {
		showAlert('Password updated successfully', 'success');
		closeChangePasswordModal();
	} else {
		showAlert(data.error || 'Failed to update password', 'error');
	}
}

// Load DSNs on page load
document.addEventListener('DOMContentLoaded', checkAuth);

// Load all DSNs
async function loadDSNs() {
	try {
		const [dsnRes, bridgeRes] = await Promise.all([
			fetch(`${API_BASE}/dsns`),
			fetch(`${API_BASE}/bridges`)
		]);
		const dsnData = await dsnRes.json();
		const bridgeData = await bridgeRes.json();

		const dsns = dsnData.dsns || [];
		const bridges = (bridgeData.bridges || []).map(b => ({
			id: b.id,
			name: b.name,
			driver: 'sqlite-bridge',
			created_at: b.created_at,
			_bridge: true,
			_connected: b.connected,
			_secret: b.secret,
		}));

		const all = [...dsns, ...bridges];

		if (all.length > 0) {
			renderDSNList(all);
		} else {
			document.getElementById('dsn-list').innerHTML = `
                <div style="text-align: center; padding: 40px; color: #999;">
                    No DSNs configured. Click "Add DSN" to get started.
                </div>
            `;
		}
	} catch (error) {
		showAlert('Failed to load DSNs: ' + error.message, 'error');
	}
}

// Render DSN list
function renderDSNList(dsns) {
	const rows = dsns.map(dsn => {
		if (dsn._bridge) {
			const statusIcon = dsn._connected ? '&#x1F7E9;' : '&#x1F7E5;';
			const statusTitle = dsn._connected ? 'Connected' : 'Disconnected';
			return `
				<tr>
					<td>${escapeHtml(dsn.name)}</td>
					<td><span class="badge badge-sqlite-bridge">sqlite-bridge</span></td>
					<td>${formatDate(dsn.created_at)}</td>
					<td>
						<button class="btn btn-primary icon-btn" onclick="openBridgeTipsModal('${escapeHtml(dsn.name)}', '${escapeHtml(dsn._secret || '')}')" title="Setup tips">&#x1F6C8;</button>
						<button class="icon-btn" style="background:none;border:none;cursor:default;" title="${statusTitle}" disabled>${statusIcon}</button>
						<button class="btn btn-primary icon-btn" onclick="openBridgeEditModal('${escapeHtml(dsn.name)}')" title="Edit">&#128221;</button>
						<button class="btn btn-danger icon-btn" onclick="deleteBridge('${escapeHtml(dsn.name)}')" title="Delete">&#10060;</button>
					</td>
				</tr>`;
		}
		const infoBtn = (dsn.driver === 'mysql' || dsn.driver === 'postgres' || dsn.driver === 'sqlite')
			? `<button class="btn btn-primary icon-btn" onclick="openDSNInfoModal('${escapeHtml(dsn.driver)}', '${escapeHtml(dsn.dsn)}')" title="Connection info">&#x1F6C8;</button>`
			: '';
		return `
			<tr>
				<td>${escapeHtml(dsn.name)}</td>
				<td><span class="badge badge-${dsn.driver}">${dsn.driver}</span></td>
				<td>${formatDate(dsn.created_at)}</td>
				<td>
					${infoBtn}
					<button class="btn btn-success icon-btn" onclick="testConnection('${dsn.id}', event, '${dsn.name}')" title="Test connection">&#128269;</button>
					<button class="btn btn-primary icon-btn" onclick="openEditModal('${dsn.id}', '${escapeHtml(dsn.name)}', '${dsn.driver}', '${escapeHtml(dsn.dsn)}')" title="Edit">&#128221;</button>
					<button class="btn btn-danger icon-btn" onclick="deleteDSN('${dsn.id}')" title="Delete">&#10060;</button>
				</td>
			</tr>`;
	}).join('');

	document.getElementById('dsn-list').innerHTML = `
		<table>
			<thead>
				<tr>
					<th>Name</th>
					<th>Driver</th>
					<th>Created</th>
					<th>Actions</th>
				</tr>
			</thead>
			<tbody>${rows}</tbody>
		</table>
	`;
}

// Open add modal
function openAddModal() {
	document.getElementById('modal-title').textContent = 'Add DSN';
	document.getElementById('dsn-id').value = '';
	document.getElementById('dsn-form').reset();
	document.getElementById('dsn-modal').classList.add('active');
}

// Open edit modal
function openEditModal(id, name, driver, dsn) {
	document.getElementById('modal-title').textContent = 'Edit DSN';
	document.getElementById('dsn-id').value = id;
	document.getElementById('dsn-name').value = name;
	document.getElementById('dsn-driver').value = driver;
	document.getElementById('dsn-string').value = dsn;
	document.getElementById('dsn-modal').classList.add('active');
	handleDriverChange();
}

// Close modal
function closeModal() {
	document.getElementById('dsn-modal').classList.remove('active');
}

// Handle form submit
async function handleSubmit(event) {
	event.preventDefault();

	const id = document.getElementById('dsn-id').value;
	const name = document.getElementById('dsn-name').value;
	const driver = document.getElementById('dsn-driver').value;
	const dsn = document.getElementById('dsn-string').value;

	const data = { name, driver, dsn };

	try {
		let response;
		if (id) {
			// Update
			response = await fetch(`${API_BASE}/dsns?id=${id}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(data)
			});
		} else {
			// Create
			response = await fetch(`${API_BASE}/dsns`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(data)
			});
		}

		const result = await response.json();

		if (response.ok && result.success) {
			showAlert(id ? 'DSN updated successfully' : 'DSN added successfully', 'success');
			closeModal();
			loadDSNs();
		} else {
			showAlert(result.error || 'Operation failed', 'error');
		}
	} catch (error) {
		showAlert('Operation failed: ' + error.message, 'error');
	}
}

// Delete DSN
async function deleteDSN(id) {
	if (!confirm('Are you sure you want to delete this DSN?')) {
		return;
	}

	try {
		const response = await fetch(`${API_BASE}/dsns?id=${id}`, {
			method: 'DELETE'
		});

		const result = await response.json();

		if (response.ok && result.success) {
			showAlert('DSN deleted successfully', 'success');
			loadDSNs();
		} else {
			showAlert(result.error || 'Delete failed', 'error');
		}
	} catch (error) {
		showAlert('Delete failed: ' + error.message, 'error');
	}
}

// Test connection
async function testConnection(id, event, name) {
	const btn = event.target;
	const originalHTML = btn.innerHTML;
	btn.innerHTML = '\u23F3';
	btn.disabled = true;

	try {
		const response = await fetch(`${API_BASE}/dsns/${id}/test`, {
			method: 'POST'
		});

		const result = await response.json();

		if (result.success) {
			showAlert(`<strong>${name}</strong>: Connection successful (${result.duration})`, 'success');
		} else {
			showAlert(`<strong>${name}</strong>: Connection failed: ${result.error}`, 'error');
		}

	} catch (error) {
		showAlert('Test failed: ' + error.message, 'error');
	} finally {
		btn.innerHTML = originalHTML;
		btn.disabled = false;
	}
}

// Parse DSN into connection info fields
function parseDSNInfo(driver, dsn) {
	let network = '', user = '', hostport = '', dbname = '', params = {};
	try {
		if (driver === 'mysql') {
			// user:pass@tcp(host:port)/dbname?params
			const m = dsn.match(/^([^:@]*)(?::[^@]*)?@([^(]+)\(([^)]*)\)\/([^?]*)(?:\?(.*))?$/);
			if (m) {
				user = m[1];
				network = m[2];
				hostport = m[3];
				dbname = m[4];
				if (m[5]) new URLSearchParams(m[5]).forEach((v, k) => params[k] = v);
			}
		} else if (driver === 'sqlite') {
			// DSN is just a file path
			network = 'file';
			dbname = dsn;
		} else if (driver === 'postgres') {
			if (dsn.startsWith('postgres://') || dsn.startsWith('postgresql://')) {
				const url = new URL(dsn);
				network = 'tcp';
				user = url.username;
				hostport = url.port ? `${url.hostname}:${url.port}` : url.hostname;
				dbname = url.pathname.replace(/^\//, '').split('?')[0];
				url.searchParams.forEach((v, k) => params[k] = v);
			} else {
				// key=value format
				const knownKeys = ['host', 'port', 'user', 'password', 'dbname', 'sslmode'];
				const get = (key) => { const m = dsn.match(new RegExp(`(?:^|\\s)${key}=(?:'([^']*)'|(\\S*))`)); return m ? (m[1] ?? m[2]) : ''; };
				network = 'tcp';
				user = get('user');
				const host = get('host') || 'localhost';
				const port = get('port') || '5432';
				hostport = `${host}:${port}`;
				dbname = get('dbname');
				for (const k of Object.keys(dsn.match(/(\w+)=/g)?.map(s => s.slice(0,-1)) ?? [])) {
					if (!knownKeys.includes(k)) params[k] = get(k);
				}
				const sslmode = get('sslmode');
				if (sslmode) params['sslmode'] = sslmode;
			}
		}
	} catch (e) { /* parse failed, leave empty */ }
	return { network, user, hostport, dbname, params };
}

function openDSNInfoModal(driver, dsn) {
	const info = parseDSNInfo(driver, dsn);
	const rowStyle = 'padding:6px 12px 6px 0;color:#666;white-space:nowrap;';
	const valStyle = 'padding:6px 0;font-weight:500;';
	const fields = driver === 'sqlite'
		? [['File', info.dbname || '—']]
		: [
			['Network', info.network || '—'],
			['User', info.user || '—'],
			['Host:Port', info.hostport || '—'],
			['DB Name', info.dbname || '—'],
		];
	let rows = fields.map(([k, v]) => `<tr><td style="${rowStyle}">${k}</td><td style="${valStyle}">${escapeHtml(v)}</td></tr>`).join('');

	const paramEntries = Object.entries(info.params);
	if (paramEntries.length > 0) {
		rows += `<tr><td colspan="2" style="padding:10px 0 4px;color:#999;font-size:12px;text-transform:uppercase;letter-spacing:0.05em;">Extra Parameters</td></tr>`;
		rows += paramEntries.map(([k, v]) =>
			`<tr><td style="${rowStyle}">${escapeHtml(k)}</td><td style="${valStyle}">${escapeHtml(v)}</td></tr>`
		).join('');
	}

	document.getElementById('dsn-info-body').innerHTML = `<table style="width:100%;border-collapse:collapse;">${rows}</table>`;
	document.getElementById('dsn-info-modal').classList.add('active');
}

function closeDSNInfoModal() {
	document.getElementById('dsn-info-modal').classList.remove('active');
}

// Show alert
function showAlert(message, type) {
	const container = document.getElementById('alert-container');
	const alert = document.createElement('div');
	alert.className = `alert alert-${type}`;
	alert.innerHTML = message;
	container.appendChild(alert);

	setTimeout(() => {
		alert.remove();
	}, 5000);
}

// Escape HTML
function escapeHtml(text) {
	const div = document.createElement('div');
	div.textContent = text;
	return div.innerHTML;
}

// Format date
function formatDate(dateString) {
	const date = new Date(dateString);
	return date.toLocaleDateString('en-US', {
		year: 'numeric',
		month: 'short',
		day: 'numeric',
		hour: '2-digit',
		minute: '2-digit'
	});
}

// Close modal on outside click
document.getElementById('dsn-modal').addEventListener('click', (e) => {
	if (e.target.id === 'dsn-modal') {
		closeModal();
	}
});

// Handle driver change - show/hide bridge fields and update hint
function handleDriverChange() {
	const driver = document.getElementById('dsn-driver').value;
	const dsnStringGroup = document.getElementById('dsn-string-group');
	const bridgeSecretGroup = document.getElementById('bridge-secret-group');
	const dsnStringInput = document.getElementById('dsn-string');
	const hintEl = document.getElementById('dsn-hint');
	const hintBody = document.getElementById('dsn-hint-body');

	if (driver === 'sqlite-bridge') {
		dsnStringGroup.style.display = 'none';
		bridgeSecretGroup.style.display = 'block';
		dsnStringInput.required = false;
	} else {
		dsnStringGroup.style.display = 'block';
		bridgeSecretGroup.style.display = 'none';
		dsnStringInput.required = true;
	}

	const hints = {
		mysql: {
			placeholder: 'user:pass@tcp(host:3306)/dbname',
			format: '<b>Format:</b> <code>user:password@tcp(host:port)/dbname?param=value</code>',
			examples: [
				'root:secret@tcp(localhost:3306)/mydb',
				'app_user:pass@tcp(db.example.com:3306)/production?charset=utf8mb4&parseTime=true',
			],
		},
		postgres: {
			placeholder: 'postgres://user:pass@host:5432/dbname',
			format: '<b>Format:</b> URL <code>postgres://user:password@host:port/dbname?sslmode=disable</code><br>or key=value: <code>host=... user=... password=... dbname=...</code>',
			examples: [
				'postgres://admin:secret@localhost:5432/mydb?sslmode=disable',
				'host=db.example.com port=5432 user=app password=pass dbname=prod sslmode=require',
			],
		},
		sqlite: {
			placeholder: '/path/to/database.db',
			format: '<b>Format:</b> absolute or relative file path to the SQLite database',
			examples: [
				'/data/myapp.db',
				'./local.db',
			],
		},
	};

	const saveBtn = document.getElementById('dsn-save-btn');
	if (saveBtn) saveBtn.disabled = !driver;

	const hint = hints[driver];
	if (hint && hintEl && hintBody) {
		const exampleLines = hint.examples.map(e => `<code style="display:block;margin:2px 0;color:#c7254e;background:#f9f2f4;padding:2px 5px;border-radius:3px;">${escapeHtml(e)}</code>`).join('');
		hintBody.innerHTML = `<div style="margin-bottom:6px;">${hint.format}</div><div style="color:#888;font-size:12px;margin-bottom:4px;">Examples:</div>${exampleLines}`;
		if (dsnStringInput) dsnStringInput.placeholder = hint.placeholder;
		hintEl.style.display = 'block';
	} else if (hintEl) {
		hintEl.style.display = 'none';
	}
}

// Generate a random secret for bridge
function generateSecret() {
	const array = new Uint8Array(32);
	crypto.getRandomValues(array);
	const secret = Array.from(array).map(b => b.toString(16).padStart(2, '0')).join('');
	document.getElementById('bridge-secret').value = secret;
}

// Open bridge edit modal
function openBridgeEditModal(name) {
	document.getElementById('bridge-edit-old-name').value = name;
	document.getElementById('bridge-edit-name').value = name;
	document.getElementById('bridge-edit-secret').value = '';
	document.getElementById('bridge-edit-modal').classList.add('active');
}

function closeBridgeEditModal() {
	document.getElementById('bridge-edit-modal').classList.remove('active');
}

function generateEditSecret() {
	const array = new Uint8Array(32);
	crypto.getRandomValues(array);
	document.getElementById('bridge-edit-secret').value =
		Array.from(array).map(b => b.toString(16).padStart(2, '0')).join('');
}

async function saveBridgeEdit() {
	const oldName = document.getElementById('bridge-edit-old-name').value;
	const name = document.getElementById('bridge-edit-name').value.trim();
	const secret = document.getElementById('bridge-edit-secret').value.trim();

	if (!name || !secret) {
		showAlert('Name and secret are required', 'error');
		return;
	}

	try {
		const response = await fetch(`${API_BASE}/bridges?name=${encodeURIComponent(oldName)}`, {
			method: 'PUT',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ name, secret })
		});
		const result = await response.json();
		if (response.ok && result.success) {
			closeBridgeEditModal();
			loadDSNs();
			openBridgeTipsModal(name, secret);
		} else {
			showAlert(result.error || 'Update failed', 'error');
		}
	} catch (error) {
		showAlert('Update failed: ' + error.message, 'error');
	}
}

// Delete a bridge
async function deleteBridge(name) {
	if (!confirm(`Are you sure you want to delete bridge "${name}"?`)) return;
	try {
		const response = await fetch(`${API_BASE}/bridges?name=${encodeURIComponent(name)}`, {
			method: 'DELETE'
		});
		const result = await response.json();
		if (response.ok && result.success) {
			showAlert('Bridge deleted successfully', 'success');
			loadDSNs();
		} else {
			showAlert(result.error || 'Delete failed', 'error');
		}
	} catch (error) {
		showAlert('Delete failed: ' + error.message, 'error');
	}
}

// Open bridge tips modal
function openBridgeTipsModal(name, secret) {
	const unidbUrl = window.location.origin;
	const secretPlaceholder = secret || '<your-bridge-secret>';
	document.getElementById('tips-binary').textContent =
		`unidb-sqlite-bridge \\\n  -name "${name}" \\\n  -file /path/to/your/database.db \\\n  -unidb ${unidbUrl} \\\n  -secret "${secretPlaceholder}"`;
	document.getElementById('tips-docker').textContent =
		`docker run \\\n  -v /path/to/your/database.db:/data/sqlite.db:ro \\\n  -e BRIDGE_NAME=${name} \\\n  -e BRIDGE_SECRET=${secretPlaceholder} \\\n  -e UNIDB_URL=${unidbUrl} \\\n  unidb-bridge`;
	document.getElementById('bridge-tips-modal').classList.add('active');
}

function closeBridgeTipsModal() {
	document.getElementById('bridge-tips-modal').classList.remove('active');
}

// Override openAddModal to handle driver change
const originalOpenAddModal = openAddModal;
openAddModal = function () {
	originalOpenAddModal();
	handleDriverChange();
}

// Override handleSubmit to handle bridge registration
const originalHandleSubmit = handleSubmit;
handleSubmit = async function (event) {
	event.preventDefault();

	const id = document.getElementById('dsn-id').value;
	const name = document.getElementById('dsn-name').value;
	const driver = document.getElementById('dsn-driver').value;
	const dsn = document.getElementById('dsn-string').value;
	const secret = document.getElementById('bridge-secret').value;

	// Handle SQLite Bridge registration
	if (driver === 'sqlite-bridge') {
		if (!secret) {
			showAlert('Please generate a secret for the bridge', 'error');
			return;
		}

		try {
			// Register the bridge
			const response = await fetch(`${API_BASE}/bridges/register`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({
					name: name,
					secret: secret,
					type: 'sqlite'
				})
			});

			const result = await response.json();

			if (response.ok && result.success) {
				closeModal();
				loadDSNs();
				openBridgeTipsModal(name, secret);
			} else {
				showAlert(result.error || 'Failed to register bridge', 'error');
			}
		} catch (error) {
			showAlert('Failed to register bridge: ' + error.message, 'error');
		}
		return;
	}

	// Regular DSN handling
	if (!dsn) {
		showAlert('Connection string is required', 'error');
		return;
	}

	const data = { name, driver, dsn };

	try {
		let response;
		if (id) {
			// Update
			response = await fetch(`${API_BASE}/dsns?id=${id}`, {
				method: 'PUT',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(data)
			});
		} else {
			// Create
			response = await fetch(`${API_BASE}/dsns`, {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(data)
			});
		}

		const result = await response.json();

		if (response.ok && result.success) {
			showAlert(id ? 'DSN updated successfully' : 'DSN added successfully', 'success');
			closeModal();
			loadDSNs();
		} else {
			showAlert(result.error || 'Operation failed', 'error');
		}
	} catch (error) {
		showAlert('Operation failed: ' + error.message, 'error');
	}
}
