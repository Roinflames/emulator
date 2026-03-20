async function loadAppVersion() {
    let version = 'dev';
    try {
        const res = await fetch('/version.json', { cache: 'no-store' });
        if (res.ok) {
            const data = await res.json();
            if (data && typeof data.version === 'string' && data.version.trim() !== '') {
                version = data.version.trim();
            }
        }
    } catch (_) {
        // Keep fallback version.
    }

    document.querySelectorAll('[data-app-version]').forEach((el) => {
        el.textContent = version;
    });
}

document.addEventListener('DOMContentLoaded', () => {
    loadAppVersion();
});
