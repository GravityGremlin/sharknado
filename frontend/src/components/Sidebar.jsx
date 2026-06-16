import React from 'react';

export default function Sidebar({ activeView, onNavigate, activePlaylistId }) {
  // Placeholder playlists for nav
  const playlists = [
    { id: 'pl-1', name: 'Favorites' },
    { id: 'pl-2', name: 'Recently Added' },
  ];

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
          className={`sidebar-item ${activeView === 'downloads' ? 'active' : ''}`}
          onClick={() => onNavigate('downloads')}
        >
          Downloads
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
        {playlists.map(pl => (
          <div
            key={pl.id}
            className={`sidebar-playlist-item ${activeView === 'playlist' && activePlaylistId === pl.id ? 'active' : ''}`}
            onClick={() => onNavigate('playlist', pl.id)}
          >
            {pl.name}
          </div>
        ))}
        {playlists.length === 0 && (
          <div style={{ padding: '4px 16px 4px 32px', fontSize: '0.75rem', color: 'var(--text3)' }}>
            No playlists yet
          </div>
        )}
      </div>
    </nav>
  );
}
