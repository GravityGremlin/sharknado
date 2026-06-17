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

export default function SearchView({ player, onDownloadStarted }) {
  const [query, setQuery] = useState('');
  const [groupedData, setGroupedData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [services, setServices] = useState({ tidal: true, qobuz: true });
  const [searched, setSearched] = useState(false);
  const [error, setError] = useState(null);
  const [expandedArtists, setExpandedArtists] = useState({});

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
      // Expand all artist sections by default
      const expanded = {};
      (data.grouped || []).forEach(g => { expanded[g.artist] = true; });
      setExpandedArtists(expanded);
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
    const url = track.url || (track.provider_id 
      ? buildProviderURL(track) 
      : buildAlbumUrl(track.provider, track.album_id));
    if (!url) return;
    try {
      await submitDownload(url, 'standard');
      if (onDownloadStarted) onDownloadStarted();
    } catch (err) {
      console.error('Track download failed:', err);
    }
  }, [onDownloadStarted]);

  const toggleArtist = useCallback((artist) => {
    setExpandedArtists(prev => ({ ...prev, [artist]: !prev[artist] }));
  }, []);

  const handleDownloadAlbum = useCallback((album) => {
    const url = buildAlbumUrl(album.provider, album.album_id);
    if (!url) return;
    submitDownload(url, 'standard')
      .then(() => {
        if (onDownloadStarted) onDownloadStarted({
          album: album.album,
          artist: album.artist,
          provider: album.provider,
        });
      })
      .catch(err => console.error('Album download failed:', err));
  }, [onDownloadStarted]);

  return (
    <div>
      <div className="view-header">
        <h2>Search</h2>
        <p>Find music across Tidal, Qobuz, and Deezer</p>
      </div>

      <div className="search-row">
        <input
          type="text"
          placeholder="Search for tracks, albums, or artists..."
          value={query}
          onChange={e => setQuery(e.target.value)}
          onKeyDown={handleKeyDown}
          autoFocus
        />
        <button className="btn btn-accent" onClick={doSearch} disabled={loading || !query.trim()}>
          {loading ? <span className="spinner" /> : 'Search'}
        </button>
      </div>

      <div className="service-toggles">
        {Object.entries(services).map(([svc, active]) => (
          <button
            key={svc}
            className={active ? 'active' : ''}
            onClick={() => toggleService(svc)}
          >
            {svc}
          </button>
        ))}
      </div>

      {error && (
        <div style={{ padding: '8px 12px', background: 'rgba(255,82,82,0.1)', border: '1px solid var(--red)', borderRadius: 'var(--radius)', marginBottom: 12, color: 'var(--red)', fontSize: '0.82rem' }}>
          {error}
        </div>
      )}

      {loading && (
        <div className="empty-state">
          <span className="spinner spinner-lg" />
          <p style={{ marginTop: 12 }}>Searching...</p>
        </div>
      )}

      {!loading && searched && groupedData.length === 0 && (
        <div className="empty-state">
          <div className="icon">&#9835;</div>
          <h3>No results found</h3>
          <p>Try a different search term or enable more services.</p>
        </div>
      )}

      {!loading && !searched && (
        <div className="empty-state">
          <div className="icon">&#9835;</div>
          <h3>Search across Tidal, Qobuz, and Deezer</h3>
          <p>Enter a query above to find music from all your streaming services.</p>
        </div>
      )}

      {groupedData.length > 0 && (
        <div className="grouped-results">
          {groupedData.map(artistGroup => (
            <div key={artistGroup.artist} className="artist-section">
              <div className="artist-header" onClick={() => toggleArtist(artistGroup.artist)}>
                <span className="expand-icon">{expandedArtists[artistGroup.artist] ? '▾' : '▸'}</span>
                <h3>{artistGroup.artist}</h3>
                <span className="artist-track-count">
                  {countArtistTracks(artistGroup.albums)} track{countArtistTracks(artistGroup.albums) !== 1 ? 's' : ''}
                </span>
              </div>
              {expandedArtists[artistGroup.artist] && (
                <div className="artist-body">
                  {artistGroup.albums.map((album, index) => (
                    <div key={`${album.provider}-${album.album_id || index}`} className="album-card">
                      <div className="album-card-header">
                        {album.cover_url ? (
                          <img
                            className="album-cover"
                            src={album.cover_url}
                            alt={album.album}
                            onError={e => { e.target.style.display = 'none'; }}
                          />
                        ) : (
                          <div className="album-cover album-cover-placeholder">♪</div>
                        )}
                        <div className="album-info">
                          <h4 className="album-name">{album.album}</h4>
                          <div className="album-meta">
                            <span className={`status-badge ${album.provider}`}>{album.provider}</span>
                            <span className="album-track-count">
                              {album.tracks.length} track{album.tracks.length !== 1 ? 's' : ''}
                            </span>
                          </div>
                        </div>
                        <div className="album-actions">
                          <button
                            className="btn btn-sm btn-accent"
                            onClick={(e) => { e.stopPropagation(); handleDownloadAlbum(album); }}
                          >
                            Download Album
                          </button>
                        </div>
                      </div>
                      <table className="track-table album-tracks">
                        <thead>
                          <tr>
                            <th style={{ width: 36 }}>#</th>
                            <th>Title</th>
                            <th style={{ width: 60 }}>Dur</th>
                            <th style={{ width: 80 }}>Actions</th>
                          </tr>
                        </thead>
                        <tbody>
                          {album.tracks.map((track, i) => (
                            <TrackRow
                              key={track.id || i}
                              track={track}
                              index={i}
                              player={player}
                              onDownload={handleDownloadTrack}
                              compact
                            />
                          ))}
                        </tbody>
                      </table>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
