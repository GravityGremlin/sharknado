import { useState, useCallback, useRef, useEffect } from 'react';
import { Howl } from 'howler';
import { getStreamURL } from '../api/client';

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
      if (howlerRef.current) {
        howlerRef.current.unload();
      }
      
      const sound = new Howl({
        src: [getStreamURL(track.id)],
        format: ['opus', 'mp3', 'flac'],
        html5: true,
        volume: volume,
        onplay: () => setIsPlaying(true),
        onpause: () => setIsPlaying(false),
        onend: () => {
          setIsPlaying(false);
          setProgress(0);
          next();
        },
        onload: () => {
          setDuration(sound.duration());
        },
        onloaderror: (_, err) => {
          console.error('Audio load failed:', err);
          setIsPlaying(false);
        },
        onplayerror: (_, err) => {
          console.error('Audio play failed:', err);
          setIsPlaying(false);
        }
      });

      howlerRef.current = sound;
      setCurrentTrack(track);
      setQueueIndex(-1);
      sound.play();
      return;
    }

    if (howlerRef.current) {
      howlerRef.current.play();
    }
  }, [volume]);

  const pause = useCallback(() => {
    if (howlerRef.current) {
      howlerRef.current.pause();
    }
    setIsPlaying(false);
  }, []);

  const togglePlay = useCallback(() => {
    if (isPlaying) {
      pause();
    } else {
      play(currentTrack);
    }
  }, [isPlaying, currentTrack, pause, play]);

  const next = useCallback(() => {
    if (queue.length > 0 && queueIndex < queue.length - 1) {
      const nextIdx = queueIndex + 1;
      setQueueIndex(nextIdx);
      play(queue[nextIdx]);
    }
  }, [queue, queueIndex, play]);

  const prev = useCallback(() => {
    if (queue.length > 0 && queueIndex > 0) {
      const prevIdx = queueIndex - 1;
      setQueueIndex(prevIdx);
      play(queue[prevIdx]);
    }
  }, [queue, queueIndex, play]);

  const seek = useCallback((value) => {
    if (howlerRef.current) {
      howlerRef.current.seek(value);
    }
    setProgress(value);
  }, []);

  const setVolume = useCallback((value) => {
    if (howlerRef.current) {
      howlerRef.current.volume(value);
    }
    setVolumeState(value);
  }, []);

  // Update progress bar
  useEffect(() => {
    if (!isPlaying) {
      if (progressIntervalRef.current) clearInterval(progressIntervalRef.current);
      return;
    }

    progressIntervalRef.current = setInterval(() => {
      if (howlerRef.current) {
        setProgress(howlerRef.current.seek());
      }
    }, 1000);

    return () => {
      if (progressIntervalRef.current) clearInterval(progressIntervalRef.current);
    };
  }, [isPlaying]);

  const setQueueAndPlay = useCallback((tracks, startIndex = 0) => {
    setQueue(tracks);
    setQueueIndex(startIndex);
    if (tracks[startIndex]) {
      play(tracks[startIndex]);
    }
  }, [play]);

  // Unload on unmount
  useEffect(() => {
    return () => {
      if (howlerRef.current) {
        howlerRef.current.unload();
      }
    };
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
