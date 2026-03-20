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

    // Setup EmulatorJS
    window.EJS_player = '#game';
    window.EJS_core = system;
    window.EJS_pathtodata = 'https://cdn.emulatorjs.org/stable/data/';
    window.EJS_color = '#dc0a2d';
    window.EJS_startOnLoaded = true;

    // ROM URL (public endpoint)
    window.EJS_gameUrl = API + '/roms/download?name=' + encodeURIComponent(currentSave.rom_name);

    // Pre-fetch save data from DB as Uint8Array (will be injected after game starts)
    if (currentSave.has_save_data) {
        try {
            const res = await apiRequest('/saves/' + currentSaveId);
            if (res && res.ok) {
                const blob = await res.blob();
                pendingSaveData = new Uint8Array(await blob.arrayBuffer());
                console.log('[PokemonWeb] Save data pre-fetched:', pendingSaveData.length, 'bytes');
            }
        } catch (e) {
            console.error('[PokemonWeb] Failed to load save data:', e);
        }
    }

    // Force save flush every 30 seconds
    window.EJS_fixedSaveInterval = 30000;

    // Callback when save is flushed
    window.EJS_onSaveSave = function(e) {
        console.log('[PokemonWeb] EJS_onSaveSave fired');
        if (e && e.save && e.save.length > 0) {
            uploadSaveData(e.save);
        }
    };

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

        debugFilesystem();

        // Inject pre-fetched save data into the emulator's virtual filesystem
        if (pendingSaveData) {
            const saveData = pendingSaveData;
            pendingSaveData = null; // Clear before inject to prevent restart loop
            injectSaveData(saveData);
        }

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

function getEmulatorInstance() {
    if (window.EJS_emulator) return window.EJS_emulator;
    const iframe = document.querySelector('#game iframe');
    if (iframe && iframe.contentWindow) {
        return iframe.contentWindow.EJS_emulator || null;
    }
    return null;
}

function getSavePath(fs) {
    const dirs = [
        '/data/saves/',
        '/home/web_user/retroarch/userdata/saves/',
    ];
    for (const dir of dirs) {
        try {
            const files = fs.readdir(dir);
            for (const f of files) {
                if (f === '.' || f === '..') continue;
                if (f.endsWith('.srm') || f.endsWith('.sav')) {
                    return dir + f;
                }
            }
        } catch {}
    }
    // Fallback: construct path from ROM name
    const romBase = currentSave.rom_name.replace(/\.[^.]+$/, '');
    return '/data/saves/' + romBase + '.srm';
}

function injectSaveData(saveData) {
    try {
        const emu = getEmulatorInstance();
        if (!emu || !emu.gameManager) {
            console.error('[PokemonWeb] No emulator/gameManager for save injection');
            return;
        }

        const gm = emu.gameManager;
        const fs = gm.FS || (gm.Module && gm.Module.FS);

        if (!fs) {
            console.error('[PokemonWeb] No FS available');
            return;
        }

        const savePath = getSavePath(fs);
        console.log('[PokemonWeb] Writing', saveData.length, 'bytes to', savePath);

        // Ensure directory exists
        const dir = savePath.substring(0, savePath.lastIndexOf('/'));
        try { fs.mkdirTree(dir); } catch {}

        fs.writeFile(savePath, saveData);

        // Try to reload save into the running core
        if (typeof gm.loadSaveFiles === 'function') {
            gm.loadSaveFiles();
            console.log('[PokemonWeb] Save reloaded via loadSaveFiles()');
            return;
        }

        if (gm.functions && typeof gm.functions.loadSaveFiles === 'function') {
            gm.functions.loadSaveFiles();
            console.log('[PokemonWeb] Save reloaded via functions.loadSaveFiles()');
            return;
        }

        // Fallback: restart the core so it re-reads the .srm from the virtual FS
        if (typeof gm.restart === 'function') {
            gm.restart();
            console.log('[PokemonWeb] Game restarted to load save');
            return;
        }

        if (typeof emu.restart === 'function') {
            emu.restart();
            console.log('[PokemonWeb] Game restarted via emu.restart()');
            return;
        }

        console.warn('[PokemonWeb] Save written to FS but could not reload core');
    } catch (err) {
        console.error('[PokemonWeb] Save injection failed:', err);
    }
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
            const path = getSavePath(fs);
            if (path) {
                try {
                    const data = fs.readFile(path);
                    if (data && data.length > 0) {
                        console.log('[PokemonWeb] Got save via FS:', data.length, 'bytes from', path);
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

function debugFilesystem() {
    try {
        const emu = getEmulatorInstance();
        if (!emu) { console.log('[PokemonWeb] DEBUG: No emulator instance'); return; }

        console.log('[PokemonWeb] DEBUG: Emulator keys:', Object.keys(emu));

        const gm = emu.gameManager;
        if (!gm) { console.log('[PokemonWeb] DEBUG: No gameManager'); return; }

        console.log('[PokemonWeb] DEBUG: GameManager keys:', Object.keys(gm));

        const fs = gm.FS || (gm.Module && gm.Module.FS);
        if (!fs) { console.log('[PokemonWeb] DEBUG: No FS found'); return; }

        // Recursively list files
        function listDir(path, depth) {
            if (depth > 3) return;
            try {
                const files = fs.readdir(path);
                for (const f of files) {
                    if (f === '.' || f === '..') continue;
                    const fullPath = path + f;
                    try {
                        const stat = fs.stat(fullPath);
                        const isDir = fs.isDir(stat.mode);
                        const size = isDir ? 'DIR' : stat.size + 'b';
                        console.log('[PokemonWeb] FS:', fullPath, size);
                        if (isDir) listDir(fullPath + '/', depth + 1);
                    } catch {}
                }
            } catch {}
        }

        console.log('[PokemonWeb] DEBUG: === Filesystem contents ===');
        listDir('/', 0);
        console.log('[PokemonWeb] DEBUG: === End filesystem ===');
    } catch (err) {
        console.error('[PokemonWeb] DEBUG filesystem error:', err);
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
