import React from 'react';
import TrackRow from './TrackRow';

export default function PlaylistView({ playlistId, player }) {
  // Placeholder — will be loaded from API
  const playlist = {
    id: playlistId,
    name: 'Favorites',
    description: 'Your favorite tracks',
    trackCount: 0,
    tracks: [],
  };

  return (
    <div>
      <div className="media-header">
        <div className="cover-large placeholder">♪</div>
        <div className="media-meta">
          <div className="label">Playlist</div>
          <h1>{playlist.name}</h1>
          <div className="sub">{playlist.description}</div>
          <div className="info-line">
            {playlist.trackCount || 0} tracks
          </div>
        </div>
      </div>

      {playlist.tracks.length === 0 && (
        <div className="empty-state">
          <div className="icon">♪</div>
          <h3>No tracks in this playlist</h3>
          <p>Search for music and add tracks to this playlist.</p>
        </div>
      )}

      {playlist.tracks.length > 0 && (
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
