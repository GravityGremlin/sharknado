import React from 'react';

function formatDuration(seconds) {
  if (!seconds || isNaN(seconds)) return '0:00';
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s.toString().padStart(2, '0')}`;
}

export default function TrackRow({ track, index, player }) {
  const handlePlay = () => {
    player.play(track);
  };

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
        <button className="play-btn" onClick={handlePlay} title="Play">▶</button>
        <button title="Add to playlist" onClick={() => {}}>+</button>
        <button className="dl-btn" title="Download">⬇</button>
      </td>
    </tr>
  );
}
