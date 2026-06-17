import React from 'react';
import { formatDuration } from '../utils/format';

export default function QueueView({ player }) {
  const { queue, queueIndex, currentTrack, play, setQueueAndPlay } = player;

  const handlePlayTrack = (index) => {
    if (queue[index]) {
      setQueueAndPlay(queue, index);
    }
  };

  if (!queue || queue.length === 0) {
    return (
      <div>
        <div className="view-header">
          <h2>Queue</h2>
          <p>Tracks waiting to play</p>
        </div>
        <div className="empty-state">
          <div className="icon">&#9835;</div>
          <h3>Queue is empty</h3>
          <p>Add tracks from search results or playlists to build your queue.</p>
        </div>
      </div>
    );
  }

  return (
    <div>
      <div className="view-header">
        <h2>Queue</h2>
        <p>{queue.length} track{queue.length !== 1 ? 's' : ''} waiting</p>
      </div>

      <div className="queue-list">
        {queue.map((track, index) => (
          <div
            key={track.id || index}
            className={`queue-item ${index === queueIndex ? 'current' : ''}`}
            onClick={() => handlePlayTrack(index)}
          >
            <div className="queue-item-number">
              {index === queueIndex ? (
                <span className="queue-playing">&#9654;</span>
              ) : (
                <span>{index + 1}</span>
              )}
            </div>
            <div className="queue-item-info">
              <div className="queue-item-title">{track.title || track.name}</div>
              <div className="queue-item-artist">{track.artist}</div>
            </div>
            <div className="queue-item-duration">
              {formatDuration(track.duration)}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}