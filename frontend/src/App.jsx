import React, { useState } from 'react';
import TitleBar from './components/TitleBar';
import Sidebar from './components/Sidebar';
import PlayerBar from './components/PlayerBar';
import SearchView from './components/SearchView';
import PlaylistView from './components/PlaylistView';
import DownloadQueue from './components/DownloadQueue';
import { usePlayer } from './hooks/usePlayer';

export default function App() {
  const [activeView, setActiveView] = useState('search');
  const [activePlaylistId, setActivePlaylistId] = useState(null);
  const player = usePlayer();

  const navigateTo = (view, playlistId) => {
    setActiveView(view);
    setActivePlaylistId(playlistId || null);
  };

  const renderContent = () => {
    switch (activeView) {
      case 'search':
        return <SearchView player={player} />;
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
              <div className="icon">♪</div>
              <h3>No playlists yet</h3>
              <p>Create a playlist from search results to get started.</p>
            </div>
          </div>
        );
      case 'downloads':
        return <DownloadQueue />;
      case 'library':
        return (
          <div>
            <div className="view-header">
              <h2>Library</h2>
              <p>Your downloaded music</p>
            </div>
            <div className="empty-state">
              <div className="icon">♫</div>
              <h3>No tracks downloaded</h3>
              <p>Download music from search results and it will appear here.</p>
            </div>
          </div>
        );
      default:
        return <SearchView player={player} />;
    }
  };

  return (
    <div className="app-layout">
      <TitleBar connected={true} />
      <Sidebar
        activeView={activeView}
        onNavigate={navigateTo}
        activePlaylistId={activePlaylistId}
      />
      <div className="main-content">
        {renderContent()}
      </div>
      <PlayerBar player={player} />
    </div>
  );
}
