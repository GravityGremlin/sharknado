import { useState, useCallback, useRef } from 'react';

export function usePlayer() {
  const [currentTrack, setCurrentTrack] = useState(null);
  const [isPlaying, setIsPlaying] = useState(false);
  const [progress, setProgress] = useState(0);
  const [duration, setDuration] = useState(0);
  const [volume, setVolumeState] = useState(0.8);
  const [queue, setQueue] = useState([]);
  const [queueIndex, setQueueIndex] = useState(-1);
  const howlerRef = useRef(null);
  const progressIntervalRef = useRef(null);

  const play = useCallback((track) => {
    if (track) {
      setCurrentTrack(track);
      setQueueIndex(-1);
    }
    setIsPlaying(true);
  }, []);

  const pause = useCallback(() => {
    setIsPlaying(false);
  }, []);

  const togglePlay = useCallback(() => {
    setIsPlaying(prev => !prev);
  }, []);

  const next = useCallback(() => {
    if (queue.length > 0 && queueIndex < queue.length - 1) {
      const nextIdx = queueIndex + 1;
      setQueueIndex(nextIdx);
      setCurrentTrack(queue[nextIdx]);
    }
  }, [queue, queueIndex]);

  const prev = useCallback(() => {
    if (queue.length > 0 && queueIndex > 0) {
      const prevIdx = queueIndex - 1;
      setQueueIndex(prevIdx);
      setCurrentTrack(queue[prevIdx]);
    }
  }, [queue, queueIndex]);

  const seek = useCallback((value) => {
    setProgress(value);
  }, []);

  const setVolume = useCallback((value) => {
    setVolumeState(value);
  }, []);

  const setQueueAndPlay = useCallback((tracks, startIndex = 0) => {
    setQueue(tracks);
    setQueueIndex(startIndex);
    setCurrentTrack(tracks[startIndex] || null);
    setIsPlaying(true);
  }, []);

  return {
    currentTrack,
    isPlaying,
    progress,
    duration,
    volume,
    queue,
    queueIndex,
    play,
    pause,
    togglePlay,
    next,
    prev,
    seek,
    setVolume,
    setQueueAndPlay,
  };
}
