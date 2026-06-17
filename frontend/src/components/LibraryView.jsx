import React, { useState, useEffect, useCallback } from 'react';
import TrackRow from './TrackRow';
import { getLibrary, scanLibrary } from '../api/client';

function formatDuration(seconds) {
  if (!seconds || isNaN(seconds)) return '0:00';
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s.toString().padStart(2, '0')}`;
}

export default function LibraryView({ player, refreshTrigger }) {
  const [tracks, setTracks] = useState([]);
  const [albums, setAlbums] = useState([]);
  const [loading, setLoading] = useState(true);
  const [scanning, setScanning] = useState(false);

  const loadLibrary = useCallback(async () => {
    try {
      const data = await getLibrary();
      setTracks(data.tracks || []);
      setAlbums(data.albums || []);
    } catch (err) {
      console.error('Load library:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    loadLibrary();
  }, [loadLibrary, refreshTrigger]);

  const handleScan = async () => {
    setScanning(true);
    try {
      await scanLibrary();
      await loadLibrary();
    } catch (err) {
      console.error('Scan failed:', err);
    } finally {
      setScanning(false);
    }
  };

  const totalDuration = tracks.reduce((sum, t) => sum + (t.duration || 0), 0);

  return (
    <div>
      <div className="view-header" style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
        <div>
          <h2>Library</h2>
          <p>
            {tracks.length} tracks, {albums.length} albums
            {totalDuration > 0 && ` \u2014 ${formatDuration(totalDuration)} total`}
          </p>
        </div>
        <button className="btn btn-sm" onClick={handleScan} disabled={scanning}>
          {scanning ? 'Scanning...' : 'Scan Downloads'}
        </button>
      </div>

      {loading && (
        <div className="empty-state">
          <span className="spinner spinner-lg" />
          <p style={{ marginTop: 12 }}>Loading library...</p>
        </div>
      )}

      {!loading && tracks.length === 0 && (
        <div className="empty-state">
          <div className="icon">&#9835;</div>
          <h3>No tracks in library</h3>
          <p>Download music or click "Scan Downloads" to import existing files.</p>
        </div>
      )}

      {!loading && tracks.length > 0 && (
        <table className="track-table">
          <thead>
            <tr>
              <th style={{ width: 40 }}>#</th>
              <th>Title</th>
              <th>Artist</th>
              <th>Album</th>
              <th style={{ width: 60 }}>Dur</th>
              <th style={{ width: 80 }}>Format</th>
              <th style={{ width: 80 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {tracks.map((track, i) => (
              <TrackRow
                key={track.id || track.path || i}
                track={track}
                index={i}
                player={player}
                onPlay={() => player.setQueueAndPlay(tracks, i)}
              />
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
