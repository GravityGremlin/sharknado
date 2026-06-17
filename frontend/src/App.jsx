import React, { useState, useCallback } from 'react';
import TitleBar from './components/TitleBar';
import Sidebar from './components/Sidebar';
import PlayerBar from './components/PlayerBar';
import SearchView from './components/SearchView';
import PlaylistView from './components/PlaylistView';
import DownloadQueue from './components/DownloadQueue';
import LibraryView from './components/LibraryView';
import { usePlayer } from './hooks/usePlayer';
import { useSSE } from './hooks/useSSE';
import { submitDownload } from './api/client';

export default function App() {
  const [activeView, setActiveView] = useState('search');
  const [activePlaylistId, setActivePlaylistId] = useState(null);
  const [downloadRefresh, setDownloadRefresh] = useState(0);
  const [libraryRefresh, setLibraryRefresh] = useState(0);
  const player = usePlayer();
  const sse = useSSE('/api/events');

  const navigateTo = (view, playlistId) => {
    setActiveView(view);
    setActivePlaylistId(playlistId || null);
  };

  const handleLibraryScanned = useCallback((result) => {
    setLibraryRefresh(t => t + 1);
    setActiveView('library');
  }, []);

  const handleDownloadStarted = useCallback(() => {
    setDownloadRefresh(t => t + 1);
    setActiveView('downloads');
  }, []);

  const renderContent = () => {
    switch (activeView) {
      case 'search':
        return <SearchView player={player} onDownloadStarted={handleDownloadStarted} />;
      case 'playlist':
        return <PlaylistView playlistId={activePlaylistId} player={player} />;
      case 'playlists':
        return (
          <div>
            <div className="view-header">
              <h2>Playlists</h2>
              <p>Your music collections</p>
            </div>
            <div className="empty-state">
              <div className="icon">&#9835;</div>
              <h3>No playlists yet</h3>
              <p>Create a playlist from search results to get started.</p>
            </div>
          </div>
        );
      case 'downloads':
        return <DownloadQueue refreshTrigger={downloadRefresh} />;
      case 'library':
        return <LibraryView player={player} refreshTrigger={libraryRefresh} />;
      default:
        return <SearchView player={player} onDownloadStarted={handleDownloadStarted} />;
    }
  };

  return (
    <div className="layout">
      <TitleBar connected={sse.connected} />
      <div className="main">
        <Sidebar
          activeView={activeView}
          onNavigate={navigateTo}
          activePlaylistId={activePlaylistId}
          onLibraryScanned={handleLibraryScanned}
        />
        <div className="main-content">
          {renderContent()}
        </div>
      </div>
      <PlayerBar player={player} />
    </div>
  );
}
