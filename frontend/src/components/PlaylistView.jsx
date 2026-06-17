import React, { useState, useEffect } from 'react';
import TrackRow from './TrackRow';
import { getPlaylist, removeTrackFromPlaylist } from '../api/client';

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

  const handlePlayTrack = (tracks, index) => {
    player.setQueueAndPlay(tracks, index);
  };

  const handleRemoveTrack = async (trackId) => {
    if (!playlistId) return;
    try {
      await removeTrackFromPlaylist(playlistId, trackId);
      setPlaylist(prev => ({
        ...prev,
        tracks: (prev.tracks || []).filter(t => t.id !== trackId)
      }));
    } catch (err) {
      console.error('Failed to remove track:', err);
    }
  };

  const tracks = playlist.tracks || [];

  return (
    <div>
      <div className="media-header">
        <div className="cover-large placeholder">♪</div>
        <div className="media-meta">
          <div className="label">Playlist</div>
          <h1>{playlist.name}</h1>
          <div className="sub">{playlist.description}</div>
          <div className="info-line">
            {tracks.length} tracks
          </div>
        </div>
      </div>

      {tracks.length === 0 && (
        <div className="empty-state">
          <div className="icon">♪</div>
          <h3>No tracks in this playlist</h3>
          <p>Search for music and add tracks to this playlist.</p>
        </div>
      )}

      {tracks.length > 0 && (
        <table className="track-table">
          <thead>
            <tr>
              <th style={{ width: 40 }}>#</th>
              <th>Title</th>
              <th>Artist</th>
              <th>Album</th>
              <th style={{ width: 60 }}>Dur</th>
              <th style={{ width: 120 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {tracks.map((track, i) => (
              <TrackRow
                key={track.id || i}
                track={track}
                index={i}
                player={player}
                compact
                onPlay={() => handlePlayTrack(tracks, i)}
              />
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
