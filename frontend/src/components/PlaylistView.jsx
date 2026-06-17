import React, { useState, useEffect } from 'react';
import TrackRow from './TrackRow';
import { getPlaylist } from '../api/client';

export default function PlaylistView({ playlistId, player }) {
  const [playlist, setPlaylist] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!playlistId) {
      setLoading(false);
      return;
    }
    setLoading(true);
    getPlaylist(playlistId)
      .then(data => setPlaylist(data))
      .catch(err => console.error('Load playlist:', err))
      .finally(() => setLoading(false));
  }, [playlistId]);

  if (loading) {
    return (
      <div className="empty-state">
        <span className="spinner spinner-lg" />
      </div>
    );
  }

  if (!playlist) {
    return (
      <div className="empty-state">
        <div className="icon">♪</div>
        <h3>Playlist not found</h3>
      </div>
    );
  }

  return (
    <div>
      <div className="media-header">
        <div className="cover-large placeholder">♪</div>
        <div className="media-meta">
          <div className="label">Playlist</div>
          <h1>{playlist.name}</h1>
          <div className="sub">{playlist.description}</div>
          <div className="info-line">
            {(playlist.tracks || []).length} tracks
          </div>
        </div>
      </div>

      {(!playlist.tracks || playlist.tracks.length === 0) && (
        <div className="empty-state">
          <div className="icon">♪</div>
          <h3>No tracks in this playlist</h3>
          <p>Search for music and add tracks to this playlist.</p>
        </div>
      )}

      {playlist.tracks && playlist.tracks.length > 0 && (
        <table className="track-table">
          <thead>
            <tr>
              <th style={{ width: 40 }}>#</th>
              <th>Title</th>
              <th>Artist</th>
              <th>Album</th>
              <th style={{ width: 60 }}>Dur</th>
              <th style={{ width: 80 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {playlist.tracks.map((track, i) => (
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
