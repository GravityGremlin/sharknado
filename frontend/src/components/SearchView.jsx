import React, { useState, useCallback } from 'react';
import TrackRow from './TrackRow';
import { search as searchAPI } from '../api/client';

export default function SearchView({ player }) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState([]);
  const [loading, setLoading] = useState(false);
  const [services, setServices] = useState({ tidal: true, qobuz: true, deezer: true });
  const [searched, setSearched] = useState(false);

  const toggleService = useCallback((svc) => {
    setServices(prev => ({ ...prev, [svc]: !prev[svc] }));
  }, []);

  const doSearch = useCallback(async () => {
    if (!query.trim()) return;
    setLoading(true);
    setSearched(true);
    try {
      const activeServices = Object.entries(services)
        .filter(([, v]) => v)
        .map(([k]) => k)
        .join(',');
      const data = await searchAPI(query, activeServices);
      setResults(data.results || []);
    } catch (err) {
      console.error('Search error:', err);
      setResults([]);
    } finally {
      setLoading(false);
    }
  }, [query, services]);

  const handleKeyDown = useCallback((e) => {
    if (e.key === 'Enter') doSearch();
  }, [doSearch]);

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

      {loading && (
        <div className="empty-state">
          <span className="spinner spinner-lg" />
          <p style={{ marginTop: 12 }}>Searching...</p>
        </div>
      )}

      {!loading && searched && results.length === 0 && (
        <div className="empty-state">
          <div className="icon">♪</div>
          <h3>No results found</h3>
          <p>Try a different search term or enable more services.</p>
        </div>
      )}

      {!loading && !searched && (
        <div className="empty-state">
          <div className="icon">♫</div>
          <h3>Search across Tidal, Qobuz, and Deezer</h3>
          <p>Enter a query above to find music from all your streaming services.</p>
        </div>
      )}

      {results.length > 0 && (
        <table className="track-table">
          <thead>
            <tr>
              <th style={{ width: 40 }}>#</th>
              <th>Title</th>
              <th>Artist</th>
              <th>Album</th>
              <th style={{ width: 60 }}>Dur</th>
              <th style={{ width: 80 }}>Src</th>
              <th style={{ width: 100 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {results.map((track, i) => (
              <TrackRow
                key={track.id || i}
                track={track}
                index={i}
                player={player}
              />
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
