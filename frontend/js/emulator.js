let currentSaveId = null;
let currentSave = null;
let saveTimeout = null;
let emulatorReady = false;

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

    // Save before leaving the page
    window.addEventListener('beforeunload', () => {
        if (emulatorReady) extractAndUploadSave();
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
    window.EJS_color = '#dc0a2d';

    // ROM URL (public endpoint, no auth needed)
    window.EJS_gameUrl = API + '/roms/download?name=' + encodeURIComponent(currentSave.rom_name);

    // Load existing save data
    if (currentSave.has_save_data) {
        const saveDataUrl = await createAuthenticatedBlobUrl('/saves/' + currentSaveId);
        if (saveDataUrl) {
            window.EJS_gameSaveUrl = saveDataUrl;
        }
    }

    // Auto-save: fires when in-game save changes (e.g. saving at Pokemon Center)
    window.EJS_onSaveUpdate = function(e) {
        console.log('[PokemonWeb] Save update detected, uploading...');
        if (saveTimeout) clearTimeout(saveTimeout);
        saveTimeout = setTimeout(() => {
            uploadSaveData(e.save);
        }, 1000);
    };

    window.EJS_onGameStart = function() {
        console.log('[PokemonWeb] Game started');
        emulatorReady = true;
        document.getElementById('loading-msg').classList.add('hidden');

        // Periodic auto-save every 30 seconds as backup
        setInterval(() => {
            extractAndUploadSave();
        }, 30000);

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

function getEmulatorInstance() {
    // EmulatorJS exposes the emulator on the window after init
    if (window.EJS_emulator) return window.EJS_emulator;

    // Try via iframe
    const iframe = document.querySelector('#game iframe');
    if (iframe && iframe.contentWindow) {
        return iframe.contentWindow.EJS_emulator || null;
    }
    return null;
}

function extractAndUploadSave() {
    try {
        const emu = getEmulatorInstance();
        if (!emu) return;

        // Try different methods to get save data
        if (emu.gameManager && typeof emu.gameManager.getSave === 'function') {
            const save = emu.gameManager.getSave();
            if (save && save.length > 0) {
                uploadSaveData(save);
                return;
            }
        }

        // Alternative: try getSaveFile
        if (typeof emu.getSaveFile === 'function') {
            const save = emu.getSaveFile();
            if (save && save.length > 0) {
                uploadSaveData(save);
                return;
            }
        }

        // Alternative: try Module FS
        if (emu.Module && emu.Module.FS) {
            try {
                const files = emu.Module.FS.readdir('/data/saves/');
                for (const file of files) {
                    if (file === '.' || file === '..') continue;
                    const data = emu.Module.FS.readFile('/data/saves/' + file);
                    if (data && data.length > 0) {
                        uploadSaveData(data);
                        return;
                    }
                }
            } catch {}
        }
    } catch (err) {
        console.error('[PokemonWeb] Extract save failed:', err);
    }
}

async function uploadSaveData(saveBuffer) {
    try {
        const res = await fetch(API + '/saves/' + currentSaveId, {
            method: 'PUT',
            headers: {
                'Authorization': 'Bearer ' + getToken()
            },
            body: saveBuffer
        });
        if (res.ok) {
            console.log('[PokemonWeb] Save uploaded successfully');
            showSaveIndicator();
        }
    } catch (err) {
        console.error('[PokemonWeb] Upload save failed:', err);
    }
}

function toggleFullscreen() {
    const container = document.querySelector('.emulator-container');
    if (!document.fullscreenElement) {
        container.requestFullscreen().catch(() => {});
    } else {
        document.exitFullscreen();
    }
}

async function manualSave() {
    const btn = document.getElementById('save-btn');
    btn.disabled = true;
    btn.textContent = 'Guardando...';

    extractAndUploadSave();

    // Give it a moment then show result
    setTimeout(() => {
        btn.textContent = 'Guardado!';
        setTimeout(() => { btn.textContent = 'Guardar'; btn.disabled = false; }, 1500);
    }, 1000);
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
