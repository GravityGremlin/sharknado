import React, { useState, useCallback, useEffect } from 'react';
import { search as searchAPI, submitDownload, getPlaylists, addTrackToPlaylist, createPlaylist } from '../api/client';

function buildProviderURL(track) {
  const id = track.provider_id || track.id?.split(':').pop();
  if (!id) return '';
  switch (track.provider) {
    case 'tidal':
      return `https://tidal.com/track/${id}`;
    case 'qobuz':
      return `https://www.qobuz.com/track/${id}`;
    case 'deezer':
      return `https://www.deezer.com/track/${id}`;
    default:
      return '';
  }
}

function flattenResults(groupedData) {
  const tracks = [];
  groupedData.forEach(group => {
    group.albums.forEach(album => {
      album.tracks.forEach(track => {
        tracks.push({
          ...track,
          artist: track.artist || album.artist || group.artist,
          album: track.album || album.album,
          album_id: track.album_id || album.album_id,
          cover_url: track.cover_url || album.cover_url,
        });
      });
    });
  });
  return tracks;
}

export default function SearchView({ player, onDownloadStarted, onPlaylistCreated }) {
  const [query, setQuery] = useState('');
  const [groupedData, setGroupedData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [services, setServices] = useState({ tidal: true, qobuz: true });
  const [searched, setSearched] = useState(false);
  const [error, setError] = useState(null);
  const [pickerTrack, setPickerTrack] = useState(null);
  const [playlists, setPlaylists] = useState([]);
  const [selectedPlaylist, setSelectedPlaylist] = useState('');
  const [newPlaylistName, setNewPlaylistName] = useState('');
  const [pickerFeedback, setPickerFeedback] = useState(null);
  const [pickerLoading, setPickerLoading] = useState(false);

  const toggleService = useCallback((svc) => {
    setServices(prev => ({ ...prev, [svc]: !prev[svc] }));
  }, []);

  const doSearch = useCallback(async () => {
    if (!query.trim()) return;
    setLoading(true);
    setSearched(true);
    setError(null);
    try {
      const activeServices = Object.entries(services)
        .filter(([, v]) => v)
        .map(([k]) => k)
        .join(',');
      const data = await searchAPI(query, activeServices);
      setGroupedData(data.grouped || []);
      if (data.error) setError(data.error);
    } catch (err) {
      console.error('Search error:', err);
      setGroupedData([]);
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, [query, services]);

  const handleKeyDown = useCallback((e) => {
    if (e.key === 'Enter') doSearch();
  }, [doSearch]);

  const handleDownloadTrack = useCallback(async (track) => {
    const url = track.url || buildProviderURL(track);
    if (!url) return;
    try {
      await submitDownload(url, 'standard');
      if (onDownloadStarted) onDownloadStarted();
    } catch (err) {
      console.error('Track download failed:', err);
    }
  }, [onDownloadStarted]);

  const openPlaylistPicker = async (track) => {
    setPickerTrack(track);
    setPickerFeedback(null);
    setSelectedPlaylist('');
    setNewPlaylistName('');
    setPickerLoading(true);
    try {
      const data = await getPlaylists();
      setPlaylists(data.playlists || []);
    } catch (err) {
      console.error('Load playlists failed:', err);
      setPickerFeedback({ type: 'error', message: 'Could not load playlists' });
    } finally {
      setPickerLoading(false);
    }
  };

  const closePlaylistPicker = useCallback(() => {
    setPickerTrack(null);
    setPickerFeedback(null);
    setSelectedPlaylist('');
    setNewPlaylistName('');
  }, []);

  useEffect(() => {
    if (!pickerTrack) return undefined;
    const handleKeyDown = (event) => {
      if (event.key === 'Escape') {
        closePlaylistPicker();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [pickerTrack, closePlaylistPicker]);

  const handleCreatePlaylist = async () => {
    if (!newPlaylistName.trim()) return;
    try {
      const playlist = await createPlaylist(newPlaylistName.trim());
      setPlaylists(prev => [...prev, playlist]);
      setSelectedPlaylist(playlist.id);
      setNewPlaylistName('');
      setPickerFeedback({ type: 'success', message: 'Playlist created' });
      if (onPlaylistCreated) onPlaylistCreated();
    } catch (err) {
      console.error('Create playlist failed:', err);
      setPickerFeedback({ type: 'error', message: 'Could not create playlist' });
    }
  };

  const handleAddToPlaylist = async () => {
    if (!pickerTrack || !selectedPlaylist) return;
    try {
      await addTrackToPlaylist(selectedPlaylist, pickerTrack.id);
      setPickerFeedback({ type: 'success', message: 'Added to playlist' });
      setTimeout(closePlaylistPicker, 700);
      if (onPlaylistCreated) onPlaylistCreated();
    } catch (err) {
      console.error('Add to playlist failed:', err);
      setPickerFeedback({ type: 'error', message: 'Could not add track' });
    }
  };

  const allTracks = flattenResults(groupedData);

  return (
    <div className="search-view">
      <div className="search-header">
        <div className="search-kicker">Find music</div>
        <div className="search-bar">
          <input
            type="text"
            placeholder="Search for tracks, albums, or artists..."
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            autoFocus
          />
          <div className="source-pills" aria-label="Search sources">
            {Object.entries(services).map(([svc, active]) => (
              <button
                key={svc}
                className={`source-pill ${active ? 'active' : ''}`}
                onClick={() => toggleService(svc)}
                type="button"
              >
                {svc.toUpperCase()}
              </button>
            ))}
          </div>
          <button className="btn btn-accent search-submit" onClick={doSearch} disabled={loading || !query.trim()} type="button">
            {loading ? <span className="spinner" /> : 'Search'}
          </button>
        </div>
        {error && <div className="search-error">{error}</div>}
      </div>

      <div className="search-results">
        {loading ? (
          <div className="loading-state">
            <span className="spinner spinner-lg" />
            <p>Searching...</p>
          </div>
        ) : searched && allTracks.length === 0 ? (
          <div className="empty-state search-empty">
            <div className="icon">?</div>
            <h3>No results found</h3>
            <p>Try a different search term or enable more sources.</p>
          </div>
        ) : allTracks.length > 0 ? (
          <>
            <div className="results-summary">
              <span>{allTracks.length} result{allTracks.length !== 1 ? 's' : ''}</span>
              <span className="results-query">{query}</span>
            </div>
            <div className="results-grid">
              {allTracks.map((track, i) => (
                <div key={track.id || i} className="result-card">
                  <div className="card-art">
                    {track.cover_url ? (
                      <img src={track.cover_url} alt="" loading="lazy" />
                    ) : (
                      <div className="card-art-placeholder">♪</div>
                    )}
                  </div>
                  <div className="card-info">
                    <div className="card-title" title={track.title || track.name}>{track.title || track.name}</div>
                    <div className="card-artist" title={track.artist}>{track.artist}</div>
                    <div className="card-album" title={track.album}>{track.album}</div>
                  </div>
                  <div className="card-actions">
                    <button className="play-btn" onClick={() => player.play(track)} title="Play" type="button">▶</button>
                    <button className="dl-btn" onClick={() => handleDownloadTrack(track)} title="Download" type="button">↓</button>
                    <button className="add-btn" onClick={() => openPlaylistPicker(track)} title="Add to playlist" type="button">+</button>
                    <span className={`track-src ${track.provider || 'unknown'}`} title={track.provider || 'unknown'} />
                  </div>
                </div>
              ))}
            </div>
          </>
        ) : (
          <div className="empty-state search-empty">
            <div className="icon">♪</div>
            <h3>Start searching</h3>
            <p>Use the search bar above to find music from Tidal and Qobuz. Results will fill this panel.</p>
          </div>
        )}
      </div>

      {pickerTrack && (
        <div className="modal-overlay" onClick={closePlaylistPicker}>
          <div className="modal playlist-modal" onClick={e => e.stopPropagation()}>
            <h3>Add to playlist</h3>
            <p className="modal-subtitle">{pickerTrack.title || pickerTrack.name}</p>
            {pickerFeedback && <div className={`playlist-picker-feedback ${pickerFeedback.type}`}>{pickerFeedback.message}</div>}
            {pickerLoading ? (
              <div className="modal-loading"><span className="spinner" /> Loading playlists...</div>
            ) : (
              <>
                <div className="form-group">
                  <label>Choose playlist</label>
                  <select value={selectedPlaylist} onChange={e => setSelectedPlaylist(e.target.value)}>
                    <option value="">Select a playlist</option>
                    {playlists.map(pl => <option key={pl.id} value={pl.id}>{pl.name}</option>)}
                  </select>
                </div>
                <div className="form-group">
                  <label>Or create new</label>
                  <div className="inline-create-row">
                    <input
                      value={newPlaylistName}
                      onChange={e => setNewPlaylistName(e.target.value)}
                      placeholder="New playlist name"
                    />
                    <button className="btn btn-sm" onClick={handleCreatePlaylist} disabled={!newPlaylistName.trim()} type="button">Create</button>
                  </div>
                </div>
                <div className="form-actions">
                  <button className="btn" onClick={closePlaylistPicker} type="button">Cancel</button>
                  <button className="btn btn-accent" onClick={handleAddToPlaylist} disabled={!selectedPlaylist} type="button">Add track</button>
                </div>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
