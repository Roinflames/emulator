document.addEventListener('DOMContentLoaded', async () => {
    if (!requireAuth()) return;

    document.getElementById('username-display').textContent = getUsername();
    document.getElementById('logout-btn').addEventListener('click', () => {
        clearAuth();
        window.location.href = '/';
    });

    await loadRoms();
    await loadSaves();

    document.getElementById('new-save-form').addEventListener('submit', createSave);
});

let availableRoms = [];

async function loadRoms() {
    const res = await apiRequest('/roms');
    if (!res) return;
    availableRoms = await res.json();

    const select = document.getElementById('rom-select');
    select.innerHTML = '<option value="">-- Selecciona una ROM --</option>';

    if (!availableRoms || availableRoms.length === 0) {
        const container = document.getElementById('roms-list');
        container.innerHTML = '<p class="empty-msg">No hay ROMs disponibles. Coloca archivos .gb, .gbc, .gba o .nds en la carpeta <code>roms/</code></p>';
        return;
    }

    const systemNames = { gb: 'Game Boy', gba: 'GBA', nds: 'NDS' };
    const container = document.getElementById('roms-list');
    container.innerHTML = '';

    availableRoms.forEach(rom => {
        // Add to select
        const opt = document.createElement('option');
        opt.value = rom.filename;
        opt.textContent = `${rom.filename} (${systemNames[rom.system] || rom.system})`;
        select.appendChild(opt);

        // Add to rom list display
        const div = document.createElement('div');
        div.className = 'rom-item';
        const sizeMB = (rom.size / 1024 / 1024).toFixed(1);
        div.innerHTML = `
            <span class="rom-system badge badge-${rom.system}">${systemNames[rom.system] || rom.system}</span>
            <span class="rom-name">${rom.filename}</span>
            <span class="rom-size">${sizeMB} MB</span>
        `;
        container.appendChild(div);
    });
}

async function loadSaves() {
    const res = await apiRequest('/saves');
    if (!res) return;
    const saves = await res.json();

    const container = document.getElementById('saves-list');
    container.innerHTML = '';

    if (!saves || saves.length === 0) {
        container.innerHTML = '<p class="empty-msg">No tienes partidas guardadas. Crea una nueva para empezar.</p>';
        return;
    }

    saves.forEach(save => {
        const div = document.createElement('div');
        div.className = 'save-item';
        const date = new Date(save.updated_at).toLocaleDateString('es');
        div.innerHTML = `
            <div class="save-info">
                <h3>${save.save_name} ${save.is_nuzlocke ? '<span class="badge badge-nuzlocke">Nuzlocke</span>' : ''}</h3>
                <p class="save-meta">${save.rom_name} &middot; ${date}</p>
            </div>
            <div class="save-actions">
                <button class="btn btn-primary btn-sm" onclick="playSave(${save.id})">Jugar</button>
                <button class="btn btn-danger btn-sm" onclick="deleteSave(${save.id}, '${save.save_name}')">Eliminar</button>
            </div>
        `;
        container.appendChild(div);
    });
}

async function createSave(e) {
    e.preventDefault();
    const romName = document.getElementById('rom-select').value;
    const saveName = document.getElementById('save-name').value;
    const isNuzlocke = document.getElementById('is-nuzlocke').checked;

    if (!romName || !saveName) return;

    const res = await apiRequest('/saves', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ rom_name: romName, save_name: saveName, is_nuzlocke: isNuzlocke })
    });

    if (res && res.ok) {
        document.getElementById('save-name').value = '';
        document.getElementById('is-nuzlocke').checked = false;
        await loadSaves();
    }
}

function playSave(saveId) {
    window.location.href = `/play.html?save=${saveId}`;
}

async function deleteSave(saveId, name) {
    if (!confirm(`Eliminar la partida "${name}"? Esta accion no se puede deshacer.`)) return;

    const res = await apiRequest(`/saves/${saveId}`, { method: 'DELETE' });
    if (res && res.ok) {
        await loadSaves();
    }
}
