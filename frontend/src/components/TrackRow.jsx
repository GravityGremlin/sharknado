import React from 'react';
import { submitDownload } from '../api/client';

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

export default function TrackRow({ track, index, player, onDownload, compact }) {
  const handlePlay = () => {
    player.play(track);
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
        <button title="Add to playlist" onClick={() => {}}>+</button>
        <button className="dl-btn" onClick={handleDownload} title="Download">&#11015;</button>
      </td>
    </tr>
  );
}
