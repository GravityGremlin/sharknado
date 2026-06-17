import React, { useState, useEffect } from 'react';
import { getPlaylists, createPlaylist, deletePlaylist } from '../api/client';

export default function PlaylistsView({ onNavigate, playlistRefresh }) {
  const [playlists, setPlaylists] = useState([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [newName, setNewName] = useState('');
  const [newDesc, setNewDesc] = useState('');

  useEffect(() => {
    loadPlaylists();
  }, [playlistRefresh]);

  const loadPlaylists = async () => {
    setLoading(true);
    try {
      const data = await getPlaylists();
      setPlaylists(data.playlists || []);
    } catch (err) {
      console.error('Failed to load playlists:', err);
    } finally {
      setLoading(false);
    }
  };

  const handleCreate = async (e) => {
    e.preventDefault();
    if (!newName.trim()) return;
    try {
      await createPlaylist(newName, newDesc);
      setNewName('');
      setNewDesc('');
      setShowCreate(false);
      // Refresh playlists after creation
      await loadPlaylists();
    } catch (err) {
      console.error('Failed to create playlist:', err);
    }
  };

  const handleDelete = async (id) => {
    if (!window.confirm('Delete this playlist?')) return;
    try {
      await deletePlaylist(id);
      // Refresh playlists after deletion
      await loadPlaylists();
    } catch (err) {
      console.error('Failed to delete playlist:', err);
    }
  };

  if (loading) {
    return (
      <div className="view">
        <div className="empty-state">
          <span className="spinner spinner-lg" />
        </div>
      </div>
    );
  }

  return (
    <div className="view">
      <div className="view-header">
        <h2>Playlists</h2>
        <p>Your music collections</p>
        <button
          className="btn btn-primary"
          onClick={() => setShowCreate(true)}
          style={{ marginTop: '8px' }}
        >
          + Create Playlist
        </button>
      </div>

      {playlists.length === 0 ? (
        <div className="empty-state">
          <div className="icon">♪</div>
          <h3>No playlists yet</h3>
          <p>Create a playlist from search results to get started.</p>
        </div>
      ) : (
        <div className="playlist-grid">
          {playlists.map(pl => (
            <div
              key={pl.id}
              className="playlist-card"
              onClick={() => onNavigate('playlist', pl.id)}
            >
              <div className="playlist-icon">♪</div>
              <div className="playlist-meta">
                <div className="playlist-name">{pl.name}</div>
                {pl.description && <div className="playlist-desc">{pl.description}</div>}
                <div className="playlist-count">{(pl.tracks || []).length} tracks</div>
              </div>
              <button
                className="btn btn-icon"
                onClick={(e) => {
                  e.stopPropagation();
                  handleDelete(pl.id);
                }}
                title="Delete playlist"
                style={{ fontSize: '1rem' }}
              >
                ×
              </button>
            </div>
          ))}
        </div>
      )}

      {showCreate && (
        <div className="modal-overlay" onClick={() => setShowCreate(false)}>
          <div className="modal" onClick={e => e.stopPropagation()}>
            <h3>Create Playlist</h3>
            <form onSubmit={handleCreate}>
              <div className="form-group">
                <label>Name</label>
                <input
                  type="text"
                  value={newName}
                  onChange={e => setNewName(e.target.value)}
                  placeholder="Enter playlist name"
                  autoFocus
                />
              </div>
              <div className="form-group">
                <label>Description</label>
                <input
                  type="text"
                  value={newDesc}
                  onChange={e => setNewDesc(e.target.value)}
                  placeholder="Optional description"
                />
              </div>
              <div className="form-actions">
                <button type="button" className="btn" onClick={() => setShowCreate(false)}>
                  Cancel
                </button>
                <button type="submit" className="btn btn-primary">
                  Create
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
