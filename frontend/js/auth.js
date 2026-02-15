const API = window.location.origin + '/api';

function getToken() {
    return localStorage.getItem('token');
}

function getUsername() {
    return localStorage.getItem('username');
}

function setAuth(token, username) {
    localStorage.setItem('token', token);
    localStorage.setItem('username', username);
}

function clearAuth() {
    localStorage.removeItem('token');
    localStorage.removeItem('username');
}

function requireAuth() {
    if (!getToken()) {
        window.location.href = '/';
        return false;
    }
    return true;
}

function authHeaders() {
    return {
        'Authorization': 'Bearer ' + getToken(),
        'Content-Type': 'application/json'
    };
}

async function apiRequest(url, options = {}) {
    if (!options.headers) {
        options.headers = {};
    }
    options.headers['Authorization'] = 'Bearer ' + getToken();

    const res = await fetch(API + url, options);
    if (res.status === 401) {
        clearAuth();
        window.location.href = '/';
        return null;
    }
    return res;
}
