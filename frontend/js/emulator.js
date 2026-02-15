let currentSaveId = null;
let currentSave = null;
let emulatorReady = false;
let pendingSaveData = null;

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

    // Pre-fetch save data if it exists
    if (currentSave.has_save_data) {
        try {
            const res = await apiRequest('/saves/' + currentSaveId);
            if (res && res.ok) {
                const blob = await res.blob();
                pendingSaveData = new Uint8Array(await blob.arrayBuffer());
                console.log('[PokemonWeb] Save data loaded:', pendingSaveData.length, 'bytes');
            }
        } catch (e) {
            console.error('[PokemonWeb] Failed to load save data:', e);
        }
    }

    // Setup EmulatorJS
    window.EJS_player = '#game';
    window.EJS_core = system;
    window.EJS_pathtodata = 'https://cdn.emulatorjs.org/stable/data/';
    window.EJS_color = '#dc0a2d';

    // ROM URL (public endpoint)
    window.EJS_gameUrl = API + '/roms/download?name=' + encodeURIComponent(currentSave.rom_name);

    // Force save flush every 30 seconds so EJS_onSaveSave fires
    window.EJS_fixedSaveInterval = 30000;

    // Callback when save is flushed (this is the correct one)
    window.EJS_onSaveSave = function(e) {
        console.log('[PokemonWeb] EJS_onSaveSave fired');
        if (e && e.save && e.save.length > 0) {
            uploadSaveData(e.save);
        }
    };

    // Also try EJS_onSaveUpdate as fallback
    window.EJS_onSaveUpdate = function(e) {
        console.log('[PokemonWeb] EJS_onSaveUpdate fired');
        if (e && e.save && e.save.length > 0) {
            uploadSaveData(e.save);
        }
    };

    window.EJS_onGameStart = function() {
        console.log('[PokemonWeb] Game started');
        emulatorReady = true;
        document.getElementById('loading-msg').classList.add('hidden');

        // Inject save data into emulator filesystem
        if (pendingSaveData) {
            injectSaveData(pendingSaveData);
        }

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

function injectSaveData(saveData) {
    try {
        const emu = getEmulatorInstance();
        if (!emu) {
            console.error('[PokemonWeb] No emulator instance found for save injection');
            return;
        }

        const gm = emu.gameManager;
        if (!gm) {
            console.error('[PokemonWeb] No gameManager found');
            return;
        }

        // Method 1: Use loadSaveFiles if available
        if (typeof gm.loadSaveFiles === 'function') {
            console.log('[PokemonWeb] Using loadSaveFiles()');
            gm.loadSaveFiles();
        }

        // Method 2: Write directly to the virtual filesystem
        if (gm.FS) {
            const savePath = getSavePath(gm.FS);
            if (savePath) {
                console.log('[PokemonWeb] Writing save to:', savePath);
                gm.FS.writeFile(savePath, saveData);
                // Reload the save into the core
                if (typeof gm.loadSave === 'function') {
                    gm.loadSave();
                }
                console.log('[PokemonWeb] Save injected! Restart the game or load save in-game.');
            }
        } else if (gm.Module && gm.Module.FS) {
            const savePath = getSavePath(gm.Module.FS);
            if (savePath) {
                console.log('[PokemonWeb] Writing save via Module.FS to:', savePath);
                gm.Module.FS.writeFile(savePath, saveData);
                console.log('[PokemonWeb] Save injected via Module.FS');
            }
        }
    } catch (err) {
        console.error('[PokemonWeb] Save injection failed:', err);
    }
}

function getSavePath(fs) {
    // Try common save paths used by RetroArch/EmulatorJS
    const paths = [
        '/data/saves/',
        '/home/web_user/retroarch/userdata/saves/',
        '/home/web_user/retroarch/userdata/states/'
    ];

    for (const dir of paths) {
        try {
            const files = fs.readdir(dir);
            for (const file of files) {
                if (file === '.' || file === '..') continue;
                if (file.endsWith('.srm') || file.endsWith('.sav')) {
                    return dir + file;
                }
            }
            // If directory exists but no save file yet, create one
            if (files.length <= 2) { // only . and ..
                // Derive save name from ROM name
                const romBase = currentSave.rom_name.replace(/\.[^.]+$/, '');
                return dir + romBase + '.srm';
            }
        } catch {}
    }
    return null;
}

function getEmulatorInstance() {
    if (window.EJS_emulator) return window.EJS_emulator;
    const iframe = document.querySelector('#game iframe');
    if (iframe && iframe.contentWindow) {
        return iframe.contentWindow.EJS_emulator || null;
    }
    return null;
}

function extractSaveData() {
    try {
        const emu = getEmulatorInstance();
        if (!emu || !emu.gameManager) return null;

        const gm = emu.gameManager;

        // Method 1: getSaveFile
        if (typeof gm.getSaveFile === 'function') {
            const save = gm.getSaveFile();
            if (save && save.length > 0) {
                console.log('[PokemonWeb] Got save via getSaveFile():', save.length, 'bytes');
                return save;
            }
        }

        // Method 2: getSave
        if (typeof gm.getSave === 'function') {
            const save = gm.getSave();
            if (save && save.length > 0) {
                console.log('[PokemonWeb] Got save via getSave():', save.length, 'bytes');
                return save;
            }
        }

        // Method 3: Read from filesystem
        const fs = gm.FS || (gm.Module && gm.Module.FS);
        if (fs) {
            const savePath = getSavePath(fs);
            if (savePath) {
                try {
                    const data = fs.readFile(savePath);
                    if (data && data.length > 0) {
                        console.log('[PokemonWeb] Got save via FS:', data.length, 'bytes from', savePath);
                        return data;
                    }
                } catch {}
            }
        }
    } catch (err) {
        console.error('[PokemonWeb] Extract save failed:', err);
    }
    return null;
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
            console.log('[PokemonWeb] Save uploaded:', saveBuffer.length, 'bytes');
            showSaveIndicator();
        } else {
            console.error('[PokemonWeb] Upload failed:', res.status);
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

    const save = extractSaveData();
    if (save && save.length > 0) {
        await uploadSaveData(save);
        btn.textContent = 'Guardado!';
    } else {
        console.warn('[PokemonWeb] No save data found to upload');
        // Log emulator state for debugging
        const emu = getEmulatorInstance();
        if (emu) {
            console.log('[PokemonWeb] Emulator keys:', Object.keys(emu));
            if (emu.gameManager) {
                console.log('[PokemonWeb] GameManager keys:', Object.keys(emu.gameManager));
            }
        }
        btn.textContent = 'Sin datos';
    }
    setTimeout(() => { btn.textContent = 'Guardar'; btn.disabled = false; }, 2000);
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
