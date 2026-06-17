import React, { useState } from 'react';
import { submitDownload, getPlaylists, addTrackToPlaylist, createPlaylist } from '../api/client';
import { formatDuration } from '../utils/format';

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

export default function TrackRow({ track, index, player, onDownload, compact, onPlay, onPlaylistCreated }) {
  const [showPlaylistPicker, setShowPlaylistPicker] = useState(false);
  const [playlists, setPlaylists] = useState([]);
  const [selectedPlaylist, setSelectedPlaylist] = useState('');
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newPlaylistName, setNewPlaylistName] = useState('');
  const [feedback, setFeedback] = useState(null);

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
      setFeedback({ type: 'success', message: 'Added to playlist' });
      setTimeout(() => {
        setFeedback(null);
        setShowPlaylistPicker(false);
      }, 1500);
    } catch (err) {
      setFeedback({ type: 'error', message: 'Failed to add track' });
      console.error('Failed to add track to playlist:', err);
    }
  };

  const handleCreatePlaylist = async () => {
    if (!newPlaylistName.trim()) return;
    try {
      const newPlaylist = await createPlaylist(newPlaylistName.trim());
      setPlaylists(prev => [...prev, newPlaylist]);
      setSelectedPlaylist(newPlaylist.id);
      setShowCreateForm(false);
      setNewPlaylistName('');
      setFeedback({ type: 'success', message: 'Playlist created' });
      if (onPlaylistCreated) onPlaylistCreated();
      setTimeout(() => setFeedback(null), 1500);
    } catch (err) {
      setFeedback({ type: 'error', message: 'Failed to create playlist' });
      console.error('Failed to create playlist:', err);
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
              {feedback && (
                <div className={`playlist-picker-feedback ${feedback.type}`}>
                  {feedback.message}
                </div>
              )}
              
              {!showCreateForm ? (
                <>
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
                    onClick={() => setShowCreateForm(true)}
                  >
                    + New
                  </button>
                  <button
                    className="btn btn-sm"
                    onClick={() => setShowPlaylistPicker(false)}
                  >
                    Cancel
                  </button>
                </>
              ) : (
                <div className="playlist-create-form">
                  <input
                    type="text"
                    placeholder="Playlist name"
                    value={newPlaylistName}
                    onChange={e => setNewPlaylistName(e.target.value)}
                    onClick={e => e.stopPropagation()}
                    autoFocus
                  />
                  <div className="playlist-create-actions">
                    <button
                      className="btn btn-primary btn-sm"
                      onClick={handleCreatePlaylist}
                      disabled={!newPlaylistName.trim()}
                    >
                      Create
                    </button>
                    <button
                      className="btn btn-sm"
                      onClick={() => {
                        setShowCreateForm(false);
                        setNewPlaylistName('');
                      }}
                    >
                      Back
                    </button>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
        <button className="dl-btn" onClick={handleDownload} title="Download">&#11015;</button>
      </td>
    </tr>
  );
}
