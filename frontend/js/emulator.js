let currentSaveId = null;
let currentSave = null;
let saveTimeout = null;

document.addEventListener('DOMContentLoaded', async () => {
    if (!requireAuth()) return;

    const params = new URLSearchParams(window.location.search);
    currentSaveId = params.get('save');
    if (!currentSaveId) {
        window.location.href = '/lobby.html';
        return;
    }

    document.getElementById('back-btn').addEventListener('click', () => {
        window.location.href = '/lobby.html';
    });

    await loadAndStartGame();
});

async function loadAndStartGame() {
    // Get save info
    const savesRes = await apiRequest('/saves');
    if (!savesRes) return;
    const saves = await savesRes.json();
    currentSave = saves.find(s => s.id === parseInt(currentSaveId));

    if (!currentSave) {
        alert('Partida no encontrada');
        window.location.href = '/lobby.html';
        return;
    }

    document.getElementById('game-title').textContent = currentSave.save_name;
    document.getElementById('rom-info').textContent = currentSave.rom_name;

    // Detect system from ROM extension
    const ext = currentSave.rom_name.split('.').pop().toLowerCase();
    const systemMap = { gb: 'gb', gbc: 'gb', gba: 'gba', nds: 'nds' };
    const system = systemMap[ext] || 'gba';

    // Setup EmulatorJS
    window.EJS_player = '#game';
    window.EJS_core = system;
    window.EJS_pathtodata = 'https://cdn.emulatorjs.org/stable/data/';

    // ROM URL (public endpoint, no auth needed)
    window.EJS_gameUrl = API + '/roms/download?name=' + encodeURIComponent(currentSave.rom_name);

    // Try to load existing save data as blob URL (needs auth)
    if (currentSave.has_save_data) {
        const saveDataUrl = await createAuthenticatedBlobUrl('/saves/' + currentSaveId);
        if (saveDataUrl) {
            window.EJS_gameSaveUrl = saveDataUrl;
        }
    }

    // Auto-save callback
    window.EJS_onSaveUpdate = function(e) {
        // Debounce saves to avoid spamming the server
        if (saveTimeout) clearTimeout(saveTimeout);
        saveTimeout = setTimeout(() => {
            uploadSaveData(e.save);
        }, 2000);
    };

    window.EJS_onGameStart = function() {
        document.getElementById('loading-msg').classList.add('hidden');
        // Show nuzlocke panel if applicable
        if (currentSave.is_nuzlocke) {
            document.getElementById('nuzlocke-panel').classList.remove('hidden');
            loadNuzlockeData();
        }
    };

    // Load EmulatorJS
    const script = document.createElement('script');
    script.src = 'https://cdn.emulatorjs.org/stable/data/loader.js';
    document.body.appendChild(script);
}

async function createAuthenticatedBlobUrl(path) {
    try {
        const res = await apiRequest(path);
        if (!res || !res.ok) return null;
        const blob = await res.blob();
        return URL.createObjectURL(blob);
    } catch {
        return null;
    }
}

async function uploadSaveData(saveBuffer) {
    try {
        await fetch(API + '/saves/' + currentSaveId, {
            method: 'PUT',
            headers: {
                'Authorization': 'Bearer ' + getToken()
            },
            body: saveBuffer
        });
        showSaveIndicator();
    } catch (err) {
        console.error('Failed to upload save:', err);
    }
}

function showSaveIndicator() {
    const indicator = document.getElementById('save-indicator');
    indicator.classList.remove('hidden');
    indicator.classList.add('show');
    setTimeout(() => {
        indicator.classList.remove('show');
        setTimeout(() => indicator.classList.add('hidden'), 300);
    }, 2000);
}
