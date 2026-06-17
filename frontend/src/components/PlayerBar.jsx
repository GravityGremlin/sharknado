import React from 'react';
import { formatDuration } from '../utils/format';

function formatTime(seconds) {
  if (!seconds || isNaN(seconds)) return '0:00';
  const m = Math.floor(seconds / 60);
  const s = Math.floor(seconds % 60);
  return `${m}:${s.toString().padStart(2, '0')}`;
}

export default function PlayerBar({ player }) {
  const {
    currentTrack,
    isPlaying,
    progress,
    duration,
    volume,
    togglePlay,
    next,
    prev,
    seek,
    setVolume,
  } = player;

  return (
    <div className="player-bar">
      <div className="track-info">
        {currentTrack?.cover_url ? (
          <img src={currentTrack.cover_url} alt="Cover" className="cover" />
        ) : (
          <div className="cover placeholder">♪</div>
        )}
        <div className="meta">
          <div className="title">
            {currentTrack ? currentTrack.title : 'No track selected'}
          </div>
          <div className="artist">
            {currentTrack ? currentTrack.artist : 'Search and play music'}
          </div>
        </div>
      </div>

      <div className="transport">
        <button onClick={prev} title="Previous">⏮</button>
        <button className="play-pause" onClick={togglePlay} title={isPlaying ? 'Pause' : 'Play'}>
          {isPlaying ? '⏸' : '▶'}
        </button>
        <button onClick={next} title="Next">⏭</button>
      </div>

      <div className="seek-section">
        <span className="time current">{formatTime(progress)}</span>
        <input
          type="range"
          className="seek-bar"
          min="0"
          max={duration || 100}
          value={progress}
          onChange={e => seek(Number(e.target.value))}
        />
        <span className="time total">{formatTime(duration)}</span>
      </div>

      <div className="volume-section">
        <span className="vol-icon">{volume === 0 ? '🔇' : volume < 0.5 ? '🔉' : '🔊'}</span>
        <input
          type="range"
          className="volume-slider"
          min="0"
          max="1"
          step="0.01"
          value={volume}
          onChange={e => setVolume(Number(e.target.value))}
        />
      </div>
    </div>
  );
}
