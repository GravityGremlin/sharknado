import React, { useState, useCallback } from 'react';
import TitleBar from './components/TitleBar';
import Sidebar from './components/Sidebar';
import PlayerBar from './components/PlayerBar';
import SearchView from './components/SearchView';
import PlaylistView from './components/PlaylistView';
import PlaylistsView from './components/PlaylistsView';
import QueueView from './components/QueueView';
import DownloadQueue from './components/DownloadQueue';
import LibraryView from './components/LibraryView';
import { usePlayer } from './hooks/usePlayer';
import { useSSE } from './hooks/useSSE';
import { submitDownload } from './api/client';

export default function App() {
  const [activeView, setActiveView] = useState(null);
  const [activePlaylistId, setActivePlaylistId] = useState(null);
  const [downloadRefresh, setDownloadRefresh] = useState(0);
  const [libraryRefresh, setLibraryRefresh] = useState(0);
  const [playlistRefresh, setPlaylistRefresh] = useState(0);
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

  const handlePlaylistCreated = useCallback(() => {
    setPlaylistRefresh(t => t + 1);
  }, []);

  const renderContent = () => {
    switch (activeView) {
      case 'playlist':
        return <PlaylistView playlistId={activePlaylistId} player={player} />;
      case 'playlists':
        return <PlaylistsView onNavigate={navigateTo} playlistRefresh={playlistRefresh} />;
      case 'queue':
        return <QueueView player={player} onNavigate={navigateTo} />;
      case 'downloads':
        return <DownloadQueue refreshTrigger={downloadRefresh} />;
      case 'library':
        return <LibraryView player={player} refreshTrigger={libraryRefresh} />;
      default:
        return <SearchView player={player} onDownloadStarted={handleDownloadStarted} onPlaylistCreated={handlePlaylistCreated} />;
    }
  };

  return (
    <div className="app-layout">
      <TitleBar connected={sse.connected} />
      <div className="main">
        <Sidebar
          activeView={activeView}
          onNavigate={navigateTo}
          activePlaylistId={activePlaylistId}
          onLibraryScanned={handleLibraryScanned}
          playlistRefresh={playlistRefresh}
          player={player}
        />
        <div className="main-content">
          {renderContent()}
        </div>
      </div>
      <PlayerBar player={player} />
    </div>
  );
}
