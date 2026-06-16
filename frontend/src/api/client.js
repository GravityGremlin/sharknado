const API_BASE = '';

async function request(path, options = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...options.headers },
    ...options,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error || err.detail || res.statusText);
  }
  if (res.status === 204) return null;
  return res.json();
}

// Search
export async function search(query, services = 'tidal,qobuz,deezer') {
  return request(`/api/search?q=${encodeURIComponent(query)}&services=${encodeURIComponent(services)}`);
}

export async function searchService(service, query) {
  return request(`/api/search/${encodeURIComponent(service)}?q=${encodeURIComponent(query)}`);
}

// Track
export async function getTrackInfo(id) {
  return request(`/api/track/${encodeURIComponent(id)}/info`);
}

export function getStreamURL(id, format = 'opus') {
  return `${API_BASE}/api/track/${encodeURIComponent(id)}/stream?format=${format}`;
}

export function getDownloadURL(id, format = 'flac') {
  return `${API_BASE}/api/track/${encodeURIComponent(id)}/download?format=${format}`;
}

// Downloads
export async function submitDownload(url, quality = 'standard') {
  return request('/api/download', {
    method: 'POST',
    body: JSON.stringify({ url, quality }),
  });
}

export async function getDownloads() {
  return request('/api/downloads');
}

export async function getDownload(id) {
  return request(`/api/downloads/${encodeURIComponent(id)}`);
}

export async function pauseDownload(id) {
  return request(`/api/downloads/${encodeURIComponent(id)}/pause`, { method: 'POST' });
}

export async function resumeDownload(id) {
  return request(`/api/downloads/${encodeURIComponent(id)}/resume`, { method: 'POST' });
}

export async function cancelDownload(id) {
  return request(`/api/downloads/${encodeURIComponent(id)}/cancel`, { method: 'POST' });
}

export async function deleteDownload(id) {
  return request(`/api/downloads/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

// Playlists
export async function getPlaylists() {
  return request('/api/playlists');
}

export async function createPlaylist(name, description = '') {
  return request('/api/playlists', {
    method: 'POST',
    body: JSON.stringify({ name, description }),
  });
}

export async function getPlaylist(id) {
  return request(`/api/playlists/${encodeURIComponent(id)}`);
}

export async function updatePlaylist(id, data) {
  return request(`/api/playlists/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(data),
  });
}

export async function deletePlaylist(id) {
  return request(`/api/playlists/${encodeURIComponent(id)}`, { method: 'DELETE' });
}

export async function addTrackToPlaylist(playlistId, trackId) {
  return request(`/api/playlists/${encodeURIComponent(playlistId)}/tracks`, {
    method: 'POST',
    body: JSON.stringify({ track_id: trackId }),
  });
}

export async function removeTrackFromPlaylist(playlistId, trackId) {
  return request(`/api/playlists/${encodeURIComponent(playlistId)}/tracks/${encodeURIComponent(trackId)}`, {
    method: 'DELETE',
  });
}

// Library
export async function getLibrary() {
  return request('/api/library');
}

export async function scanLibrary() {
  return request('/api/library/scan', { method: 'POST' });
}
