import React, { useState } from 'react';
import { submitDownload, getPlaylists, addTrackToPlaylist } from '../api/client';

function formatDuration(seconds) {
  if (!seconds || isNaN(seconds)) return '0:00';
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s.toString().padStart(2, '0')}`;
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

export default function TrackRow({ track, index, player, onDownload, compact, onPlay }) {
  const [showPlaylistPicker, setShowPlaylistPicker] = useState(false);
  const [playlists, setPlaylists] = useState([]);
  const [selectedPlaylist, setSelectedPlaylist] = useState('');

  const handlePlay = () => {
    if (onPlay) {
      onPlay();
    } else {
      player.play(track);
    }
  };

  const handleDownload = async () => {
    const url = track.url || buildProviderURL(track);
    try {
      await submitDownload(url, 'standard');
      if (onDownload) onDownload(track);
    } catch (err) {
      console.error('Download failed:', err);
    }
  };

  const handleAddToPlaylist = async (playlistId) => {
    if (!playlistId) return;
    try {
      await addTrackToPlaylist(playlistId, track.id);
      setShowPlaylistPicker(false);
    } catch (err) {
      console.error('Failed to add track to playlist:', err);
    }
  };

  const loadPlaylists = async () => {
    try {
      const data = await getPlaylists();
      setPlaylists(data.playlists || []);
    } catch (err) {
      console.error('Failed to load playlists:', err);
    }
  };

  const handlePickerClick = (e) => {
    e.stopPropagation();
    if (showPlaylistPicker) {
      setShowPlaylistPicker(false);
    } else {
      loadPlaylists();
      setShowPlaylistPicker(true);
    }
  };

  if (compact) {
    return (
      <tr>
        <td style={{ color: 'var(--text3)', fontSize: '0.75rem' }}>{index + 1}</td>
        <td>
          <span className="track-title">{track.title || track.name}</span>
        </td>
        <td className="track-duration" style={{ textAlign: 'right' }}>
          {formatDuration(track.duration)}
          <span className={`track-src-mini ${track.provider || 'unknown'}`} />
        </td>
        <td className="track-actions">
          <button className="play-btn" onClick={handlePlay} title="Play">&#9654;</button>
          <button className="dl-btn" onClick={handleDownload} title="Download">&#11015;</button>
        </td>
      </tr>
    );
  }

  return (
    <tr>
      <td style={{ color: 'var(--text3)', fontSize: '0.75rem' }}>{index + 1}</td>
      <td>
        <span className="track-title">{track.title || track.name}</span>
      </td>
      <td className="track-artist">{track.artist}</td>
      <td className="track-album">{track.album}</td>
      <td className="track-duration">{formatDuration(track.duration)}</td>
      <td>
        <span className={`status-badge ${track.provider || 'unknown'}`}>
          {track.provider || '?'}
        </span>
      </td>
      <td className="track-actions">
        <button className="play-btn" onClick={handlePlay} title="Play">&#9654;</button>
        <div className="playlist-picker-wrapper">
          <button className="add-btn" title="Add to playlist" onClick={handlePickerClick}>+</button>
          {showPlaylistPicker && (
            <div className="playlist-picker" onClick={e => e.stopPropagation()}>
              {playlists.length === 0 ? (
                <div className="playlist-picker-empty">
                  No playlists yet
                </div>
              ) : (
                <select
                  value={selectedPlaylist}
                  onChange={e => setSelectedPlaylist(e.target.value)}
                  onClick={e => e.stopPropagation()}
                >
                  <option value="">-- Select Playlist --</option>
                  {playlists.map(pl => (
                    <option key={pl.id} value={pl.id}>{pl.name}</option>
                  ))}
                </select>
              )}
              <button
                className="btn btn-primary btn-sm"
                onClick={() => handleAddToPlaylist(selectedPlaylist)}
                disabled={!selectedPlaylist}
              >
                Add
              </button>
              <button
                className="btn btn-sm"
                onClick={() => setShowPlaylistPicker(false)}
              >
                Cancel
              </button>
            </div>
          )}
        </div>
        <button className="dl-btn" onClick={handleDownload} title="Download">&#11015;</button>
      </td>
    </tr>
  );
}
