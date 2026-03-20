let nuzlockeData = [];
let nuzlockeRoutes = [];
let editingPokemonId = null;

const NUZLOCKE_RULES = [
    "Solo puedes capturar el primer Pokemon de cada ruta/zona.",
    "Si un Pokemon se debilita, se considera muerto y debe ser liberado o guardado en el PC permanentemente.",
    "Debes ponerle un apodo a cada Pokemon capturado.",
    "Si todos tus Pokemon mueren, la partida termina (whiteout = game over)."
];

async function loadNuzlockeData() {
    const [pokemonRes, routesRes] = await Promise.all([
        apiRequest('/saves/' + currentSaveId + '/nuzlocke'),
        apiRequest('/saves/' + currentSaveId + '/nuzlocke/routes')
    ]);
    if (!pokemonRes || !routesRes) return;

    nuzlockeData = await pokemonRes.json();
    nuzlockeRoutes = await routesRes.json();
    renderNuzlocke();
}

async function loadNuzlockeRoutes() {
    const res = await apiRequest('/saves/' + currentSaveId + '/nuzlocke/routes');
    if (!res) return;
    nuzlockeRoutes = await res.json();
    renderNuzlocke();
}

function renderNuzlocke() {
    renderRules();
    renderTeam();
    renderGraveyard();
    renderStats();
    renderRoutes();
}

function renderRules() {
    const container = document.getElementById('nuzlocke-rules');
    container.innerHTML = '<h3>Reglas Nuzlocke</h3><ul>' +
        NUZLOCKE_RULES.map(r => `<li>${r}</li>`).join('') +
        '</ul>';
}

function renderTeam() {
    const alive = nuzlockeData.filter(p => p.is_alive);
    const container = document.getElementById('nuzlocke-team');
    container.innerHTML = '<h3>Equipo (' + alive.length + ')</h3>';

    if (alive.length === 0) {
        container.innerHTML += '<p class="empty-msg">Sin Pokemon en el equipo</p>';
        return;
    }

    alive.forEach(p => {
        const div = document.createElement('div');
        div.className = 'nuz-pokemon alive';
        if (editingPokemonId === p.id) {
            div.innerHTML = `
                <form class="nuz-edit-form" onsubmit="savePokemonEdit(event, ${p.id})">
                    <input type="text" name="pokemon_name" placeholder="Pokemon" value="${escapeHtml(p.pokemon_name || '')}" required>
                    <input type="text" name="nickname" placeholder="Apodo" value="${escapeHtml(p.nickname || '')}">
                    <input type="text" name="route" placeholder="Ruta/Zona" value="${escapeHtml(p.route || '')}" required>
                    <input type="number" name="level_caught" placeholder="Nivel" min="1" max="100" value="${p.level_caught || ''}">
                    <input type="text" name="notes" placeholder="Notas" value="${escapeHtml(p.notes || '')}">
                    <div class="nuz-actions">
                        <button type="submit" class="btn btn-primary btn-xs">Guardar</button>
                        <button type="button" class="btn btn-xs" onclick="cancelPokemonEdit()">Cancelar</button>
                        <button type="button" class="btn btn-danger btn-xs" onclick="deletePokemon(${p.id}, '${(p.nickname || p.pokemon_name).replace(/'/g, "\\'")}')">Eliminar</button>
                    </div>
                </form>
            `;
        } else {
            div.innerHTML = `
                <div class="nuz-pokemon-info">
                    <strong>${escapeHtml(p.nickname || p.pokemon_name)}</strong>
                    ${p.nickname ? `<span class="nuz-species">(${escapeHtml(p.pokemon_name)})</span>` : ''}
                    <span class="nuz-route">Ruta: ${escapeHtml(p.route)}</span>
                    ${p.level_caught ? `<span class="nuz-level">Nv. ${p.level_caught}</span>` : ''}
                </div>
                <div class="nuz-actions">
                    <button class="btn btn-xs" onclick="startPokemonEdit(${p.id})">Editar</button>
                    <button class="btn btn-danger btn-xs" onclick="killPokemon(${p.id}, '${(p.nickname || p.pokemon_name).replace(/'/g, "\\'")}')">Muerto</button>
                    <button class="btn btn-danger btn-xs" onclick="deletePokemon(${p.id}, '${(p.nickname || p.pokemon_name).replace(/'/g, "\\'")}')">Eliminar</button>
                </div>
            `;
        }
        container.appendChild(div);
    });
}

function renderGraveyard() {
    const dead = nuzlockeData.filter(p => !p.is_alive);
    const container = document.getElementById('nuzlocke-graveyard');
    container.innerHTML = '<h3>Cementerio (' + dead.length + ')</h3>';

    if (dead.length === 0) {
        container.innerHTML += '<p class="empty-msg">Sin bajas... por ahora</p>';
        return;
    }

    dead.forEach(p => {
        const div = document.createElement('div');
        div.className = 'nuz-pokemon dead';
        if (editingPokemonId === p.id) {
            div.innerHTML = `
                <form class="nuz-edit-form" onsubmit="savePokemonEdit(event, ${p.id})">
                    <input type="text" name="pokemon_name" placeholder="Pokemon" value="${escapeHtml(p.pokemon_name || '')}" required>
                    <input type="text" name="nickname" placeholder="Apodo" value="${escapeHtml(p.nickname || '')}">
                    <input type="text" name="route" placeholder="Ruta/Zona" value="${escapeHtml(p.route || '')}" required>
                    <input type="number" name="level_caught" placeholder="Nivel" min="1" max="100" value="${p.level_caught || ''}">
                    <input type="text" name="notes" placeholder="Notas" value="${escapeHtml(p.notes || '')}">
                    <div class="nuz-actions">
                        <button type="submit" class="btn btn-primary btn-xs">Guardar</button>
                        <button type="button" class="btn btn-xs" onclick="cancelPokemonEdit()">Cancelar</button>
                        <button type="button" class="btn btn-danger btn-xs" onclick="deletePokemon(${p.id}, '${(p.nickname || p.pokemon_name).replace(/'/g, "\\'")}')">Eliminar</button>
                    </div>
                </form>
            `;
        } else {
            div.innerHTML = `
                <div class="nuz-pokemon-info">
                    <strong>${escapeHtml(p.nickname || p.pokemon_name)}</strong>
                    ${p.nickname ? `<span class="nuz-species">(${escapeHtml(p.pokemon_name)})</span>` : ''}
                    <span class="nuz-route">Ruta: ${escapeHtml(p.route)}</span>
                    ${p.cause_of_death ? `<span class="nuz-death">Causa: ${escapeHtml(p.cause_of_death)}</span>` : ''}
                </div>
                <div class="nuz-actions">
                    <button class="btn btn-xs" onclick="startPokemonEdit(${p.id})">Editar</button>
                    <button class="btn btn-danger btn-xs" onclick="deletePokemon(${p.id}, '${(p.nickname || p.pokemon_name).replace(/'/g, "\\'")}')">Eliminar</button>
                </div>
            `;
        }
        container.appendChild(div);
    });
}

function renderStats() {
    const alive = nuzlockeData.filter(p => p.is_alive).length;
    const dead = nuzlockeData.filter(p => !p.is_alive).length;
    const total = nuzlockeData.length;
    const routes = new Set(nuzlockeData.map(p => p.route)).size;
    const routeCounts = countRouteStatuses();

    const container = document.getElementById('nuzlocke-stats');
    container.innerHTML = `
        <h3>Estadisticas</h3>
        <div class="stats-grid">
            <div class="stat">
                <span class="stat-number">${total}</span>
                <span class="stat-label">Capturados</span>
            </div>
            <div class="stat">
                <span class="stat-number">${alive}</span>
                <span class="stat-label">Vivos</span>
            </div>
            <div class="stat">
                <span class="stat-number">${dead}</span>
                <span class="stat-label">Muertos</span>
            </div>
            <div class="stat">
                <span class="stat-number">${routes}</span>
                <span class="stat-label">Rutas</span>
            </div>
            <div class="stat">
                <span class="stat-number">${routeCounts.pending}</span>
                <span class="stat-label">Pendientes</span>
            </div>
        </div>
    `;
}

function renderRoutes() {
    const container = document.getElementById('nuzlocke-routes');
    if (!container) return;

    const counts = countRouteStatuses();
    const sorted = [...nuzlockeRoutes].sort((a, b) => (a.route || '').localeCompare(b.route || '', 'es'));

    container.innerHTML = `
        <h3>Rutas</h3>
        <div class="nuz-route-stats">
            <span class="nuz-pill">Pendientes: <strong>${counts.pending}</strong></span>
            <span class="nuz-pill">Visitadas: <strong>${counts.visited}</strong></span>
            <span class="nuz-pill">Capturadas: <strong>${counts.captured}</strong></span>
            <span class="nuz-pill">Perdidas: <strong>${counts.missed}</strong></span>
        </div>
        <form id="add-route-form" class="nuz-form nuz-form-row">
            <input type="text" name="route" placeholder="Ruta/Zona (ej: Ruta 2)" required>
            <select name="status">
                <option value="pending" selected>Pendiente</option>
                <option value="visited">Visitada</option>
                <option value="captured">Capturada</option>
                <option value="missed">Perdida</option>
            </select>
            <button type="submit" class="btn btn-primary btn-xs">Agregar</button>
        </form>
        <div class="nuz-route-list">
            ${sorted.length === 0 ? '<p class="empty-msg">Sin rutas aún</p>' : ''}
        </div>
    `;

    const list = container.querySelector('.nuz-route-list');
    sorted.forEach(r => {
        const div = document.createElement('div');
        const st = (r.status || 'pending').toLowerCase();
        div.className = 'nuz-route-item';
        div.innerHTML = `
            <div class="nuz-route-info">
                <strong>${escapeHtml(r.route)}</strong>
                <span class="nuz-route-status status-${st}">${escapeHtml(labelRouteStatus(st))}</span>
            </div>
            <div class="nuz-actions">
                <button class="btn btn-xs" onclick="setRouteStatus(${r.id}, 'pending')">Pendiente</button>
                <button class="btn btn-xs" onclick="setRouteStatus(${r.id}, 'visited')">Visitada</button>
                <button class="btn btn-xs" onclick="setRouteStatus(${r.id}, 'captured')">Capturada</button>
                <button class="btn btn-xs" onclick="setRouteStatus(${r.id}, 'missed')">Perdida</button>
                <button class="btn btn-danger btn-xs" onclick="deleteRoute(${r.id}, '${(r.route || '').replace(/'/g, "\\'")}')">X</button>
            </div>
        `;
        list.appendChild(div);
    });

    const form = document.getElementById('add-route-form');
    form.addEventListener('submit', async (e) => {
        e.preventDefault();
        const route = form.route.value.trim();
        const status = form.status.value;
        if (!route) return;

        const res = await apiRequest('/saves/' + currentSaveId + '/nuzlocke/routes', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ route, status })
        });
        if (res && res.ok) {
            form.reset();
            await loadNuzlockeRoutes();
        }
    });
}

function countRouteStatuses() {
    const counts = { pending: 0, visited: 0, captured: 0, missed: 0 };
    (nuzlockeRoutes || []).forEach(r => {
        const st = (r.status || 'pending').toLowerCase();
        if (counts[st] !== undefined) counts[st] += 1;
    });
    return counts;
}

function labelRouteStatus(status) {
    switch ((status || 'pending').toLowerCase()) {
        case 'visited': return 'visitada';
        case 'captured': return 'capturada';
        case 'missed': return 'perdida';
        default: return 'pendiente';
    }
}

// Add pokemon form
document.getElementById('add-pokemon-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.target;

    const pokemon = {
        pokemon_name: form.pokemon_name.value.trim(),
        nickname: form.nickname.value.trim(),
        route: form.route.value.trim(),
        level_caught: parseInt(form.level_caught.value) || 0,
        notes: ''
    };

    if (!pokemon.pokemon_name || !pokemon.route) return;

    const res = await apiRequest('/saves/' + currentSaveId + '/nuzlocke', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(pokemon)
    });

    if (res && res.ok) {
        const newPokemon = await res.json();
        nuzlockeData.push(newPokemon);
        renderNuzlocke();
        form.reset();
        loadNuzlockeRoutes();
    }
});

async function killPokemon(pokemonId, name) {
    const cause = prompt(`Como murio ${name}?`, '');
    if (cause === null) return; // cancelled

    const res = await apiRequest(`/saves/${currentSaveId}/nuzlocke/${pokemonId}/kill`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ cause_of_death: cause })
    });

    if (res && res.ok) {
        const p = nuzlockeData.find(p => p.id === pokemonId);
        if (p) {
            p.is_alive = false;
            p.cause_of_death = cause;
        }
        renderNuzlocke();
    }
}

function startPokemonEdit(pokemonId) {
    editingPokemonId = pokemonId;
    renderNuzlocke();
}

function cancelPokemonEdit() {
    editingPokemonId = null;
    renderNuzlocke();
}

async function savePokemonEdit(e, pokemonId) {
    e.preventDefault();
    const form = e.target;

    const payload = {
        pokemon_name: form.pokemon_name.value.trim(),
        nickname: form.nickname.value.trim(),
        route: form.route.value.trim(),
        level_caught: parseInt(form.level_caught.value) || 0,
        notes: form.notes.value.trim()
    };

    if (!payload.pokemon_name || !payload.route) return;

    const res = await apiRequest(`/saves/${currentSaveId}/nuzlocke/${pokemonId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
    });

    if (res && res.ok) {
        const updated = await res.json();
        const idx = nuzlockeData.findIndex(p => p.id === pokemonId);
        if (idx >= 0) nuzlockeData[idx] = updated;
        editingPokemonId = null;
        renderNuzlocke();
        loadNuzlockeRoutes();
    }
}

async function deletePokemon(pokemonId, name) {
    if (!confirm(`Eliminar a ${name}?`)) return;

    const res = await apiRequest(`/saves/${currentSaveId}/nuzlocke/${pokemonId}`, {
        method: 'DELETE'
    });
    if (res && res.ok) {
        nuzlockeData = nuzlockeData.filter(p => p.id !== pokemonId);
        if (editingPokemonId === pokemonId) editingPokemonId = null;
        renderNuzlocke();
        loadNuzlockeRoutes();
    }
}

async function setRouteStatus(routeId, status) {
    const route = nuzlockeRoutes.find(r => r.id === routeId);
    if (!route) return;

    const res = await apiRequest(`/saves/${currentSaveId}/nuzlocke/routes/${routeId}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ route: route.route, status, notes: route.notes || '' })
    });
    if (res && res.ok) {
        const updated = await res.json();
        const idx = nuzlockeRoutes.findIndex(r => r.id === routeId);
        if (idx >= 0) nuzlockeRoutes[idx] = updated;
        renderNuzlocke();
    }
}

async function deleteRoute(routeId, routeName) {
    if (!confirm(`Eliminar ruta "${routeName}"?`)) return;
    const res = await apiRequest(`/saves/${currentSaveId}/nuzlocke/routes/${routeId}`, { method: 'DELETE' });
    if (res && res.ok) {
        nuzlockeRoutes = nuzlockeRoutes.filter(r => r.id !== routeId);
        renderNuzlocke();
    }
}

function escapeHtml(str) {
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#039;');
}
