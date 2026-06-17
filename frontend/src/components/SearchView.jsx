import React, { useState, useCallback } from 'react';
import TrackRow from './TrackRow';
import { search as searchAPI, submitDownload } from '../api/client';

function buildAlbumUrl(provider, albumId) {
  switch (provider) {
    case 'qobuz':
      return `https://www.qobuz.com/album/${albumId}`;
    case 'tidal':
      return `https://tidal.com/album/${albumId}`;
    case 'deezer':
      return `https://www.deezer.com/album/${albumId}`;
    default:
      return '';
  }
}

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

function countArtistTracks(albums) {
  return albums.reduce((sum, a) => sum + a.tracks.length, 0);
}

function flattenResults(groupedData) {
  const tracks = [];
  groupedData.forEach(group => {
    group.albums.forEach(album => {
      album.tracks.forEach(track => {
        tracks.push({ ...track, album, artist: group.artist });
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

  const allTracks = flattenResults(groupedData);

  return (
    <div className="search-view">
      {/* Search Header */}
      <div className="search-header">
        <div className="search-bar">
          <input
            type="text"
            placeholder="Search for tracks, albums, or artists..."
            value={query}
            onChange={e => setQuery(e.target.value)}
            onKeyDown={handleKeyDown}
            autoFocus
          />
          <div className="source-pills">
            {Object.entries(services).map(([svc, active]) => (
              <button
                key={svc}
                className={`source-pill ${active ? 'active' : ''}`}
                onClick={() => toggleService(svc)}
              >
                {svc.toUpperCase()}
              </button>
            ))}
          </div>
        </div>
        {error && <div className="search-error">{error}</div>}
      </div>

      {/* Results */}
      <div className="search-results">
        {loading ? (
          <div className="loading-state">
            <span className="spinner spinner-lg" />
            <p>Searching...</p>
          </div>
        ) : searched && allTracks.length === 0 ? (
          <div className="empty-state search-empty">
            <div className="icon">🔍</div>
            <h3>No results found</h3>
            <p>Try a different search term or enable more sources.</p>
          </div>
        ) : allTracks.length > 0 ? (
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
                  <div className="card-title">{track.title || track.name}</div>
                  <div className="card-artist">{track.artist}</div>
                  <div className="card-album">{track.album}</div>
                </div>
                <div className="card-actions">
                  <button
                    className="play-btn"
                    onClick={() => player.play(track)}
                    title="Play"
                  >
                    ▶
                  </button>
                  <button
                    className="dl-btn"
                    onClick={() => handleDownloadTrack(track)}
                    title="Download"
                  >
                    ⬇
                  </button>
                  <button
                    className="add-btn"
                    onClick={() => player.addToPlaylist?.(track)}
                    title="Add to playlist"
                  >
                    +
                  </button>
                </div>
                <span className={`track-src ${track.provider || 'unknown'}`} />
              </div>
            ))}
          </div>
        ) : (
          <div className="empty-state search-empty">
            <div className="icon">🎵</div>
            <h3>Start searching</h3>
            <p>Enter a query above to find music from Tidal, Qobuz, and Deezer.</p>
          </div>
        )}
      </div>
    </div>
  );
}