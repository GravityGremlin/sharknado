import React, { useState, useEffect } from 'react';
import { scanLibrary, getPlaylists } from '../api/client';

export default function Sidebar({ activeView, onNavigate, activePlaylistId, onLibraryScanned, playlistRefresh, player }) {
  const [playlists, setPlaylists] = useState([]);
  const [loading, setLoading] = useState(true);
  const [scanning, setScanning] = useState(false);

  useEffect(() => {
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
    loadPlaylists();
  }, [playlistRefresh]);

  const handleScan = async () => {
    setScanning(true);
    try {
      const result = await scanLibrary();
      if (onLibraryScanned) onLibraryScanned(result);
    } catch (err) {
      console.error('Scan failed:', err);
    } finally {
      setScanning(false);
    }
  };

  return (
    <nav className="sidebar">
      <div className="sidebar-section">
        <div className="sidebar-section-title">Browse</div>
        <div
          className={`sidebar-item ${activeView === 'search' ? 'active' : ''}`}
          onClick={() => onNavigate('search')}
        >
          Search
        </div>
        <div
          className={`sidebar-item ${activeView === 'library' ? 'active' : ''}`}
          onClick={() => onNavigate('library')}
        >
          Library
        </div>
        <div
          className="sidebar-item"
          onClick={handleScan}
          style={{ fontSize: '0.75rem', color: scanning ? 'var(--accent)' : 'var(--text3)' }}
        >
          {scanning ? 'Scanning...' : 'Scan Downloads'}
        </div>
        <div
          className={`sidebar-item ${activeView === 'downloads' ? 'active' : ''}`}
          onClick={() => onNavigate('downloads')}
        >
          Downloads
        </div>
        <div
          className={`sidebar-item ${activeView === 'queue' ? 'active' : ''}`}
          onClick={() => onNavigate('queue')}
        >
          Queue<span className="count">{player?.queue?.length || 0}</span>
        </div>
      </div>

      <div className="sidebar-section">
        <div
          className="sidebar-section-title"
          onClick={() => onNavigate('playlists')}
          style={{ cursor: 'pointer' }}
        >
          Playlists
        </div>
        {loading ? (
          <div className="sidebar-playlist-item" style={{ color: 'var(--text3)' }}>
            Loading...
          </div>
        ) : playlists.length === 0 ? (
          <div style={{ padding: '4px 16px 4px 32px', fontSize: '0.75rem', color: 'var(--text3)' }}>
            No playlists yet
          </div>
        ) : (
          playlists.map(pl => (
            <div
              key={pl.id}
              className={`sidebar-playlist-item ${activeView === 'playlist' && activePlaylistId === pl.id ? 'active' : ''}`}
              onClick={() => onNavigate('playlist', pl.id)}
            >
              {pl.name}
            </div>
          ))
        )}
      </div>
    </nav>
  );
}
