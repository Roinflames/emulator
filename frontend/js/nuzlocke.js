let nuzlockeData = [];

const NUZLOCKE_RULES = [
    "Solo puedes capturar el primer Pokemon de cada ruta/zona.",
    "Si un Pokemon se debilita, se considera muerto y debe ser liberado o guardado en el PC permanentemente.",
    "Debes ponerle un apodo a cada Pokemon capturado.",
    "Si todos tus Pokemon mueren, la partida termina (whiteout = game over)."
];

async function loadNuzlockeData() {
    const res = await apiRequest('/saves/' + currentSaveId + '/nuzlocke');
    if (!res) return;
    nuzlockeData = await res.json();
    renderNuzlocke();
}

function renderNuzlocke() {
    renderRules();
    renderTeam();
    renderGraveyard();
    renderStats();
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
        div.innerHTML = `
            <div class="nuz-pokemon-info">
                <strong>${p.nickname || p.pokemon_name}</strong>
                ${p.nickname ? `<span class="nuz-species">(${p.pokemon_name})</span>` : ''}
                <span class="nuz-route">Ruta: ${p.route}</span>
                ${p.level_caught ? `<span class="nuz-level">Nv. ${p.level_caught}</span>` : ''}
            </div>
            <button class="btn btn-danger btn-xs" onclick="killPokemon(${p.id}, '${(p.nickname || p.pokemon_name).replace(/'/g, "\\'")}')">Muerto</button>
        `;
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
        div.innerHTML = `
            <div class="nuz-pokemon-info">
                <strong>${p.nickname || p.pokemon_name}</strong>
                ${p.nickname ? `<span class="nuz-species">(${p.pokemon_name})</span>` : ''}
                <span class="nuz-route">Ruta: ${p.route}</span>
                ${p.cause_of_death ? `<span class="nuz-death">Causa: ${p.cause_of_death}</span>` : ''}
            </div>
        `;
        container.appendChild(div);
    });
}

function renderStats() {
    const alive = nuzlockeData.filter(p => p.is_alive).length;
    const dead = nuzlockeData.filter(p => !p.is_alive).length;
    const total = nuzlockeData.length;
    const routes = new Set(nuzlockeData.map(p => p.route)).size;

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
        </div>
    `;
}

// Add pokemon form
document.getElementById('add-pokemon-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const form = e.target;

    const pokemon = {
        pokemon_name: form.pokemon_name.value.trim(),
        nickname: form.nickname.value.trim(),
        route: form.route.value.trim(),
        level_caught: parseInt(form.level_caught.value) || 0
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
