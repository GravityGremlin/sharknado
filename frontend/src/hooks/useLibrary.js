import { useState, useEffect, useCallback } from 'react';
import { getLibrary, getPlaylists } from '../api/client';

export function useLibrary() {
  const [tracks, setTracks] = useState([]);
  const [albums, setAlbums] = useState([]);
  const [playlists, setPlaylists] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);
  const [refreshTrigger, setRefreshTrigger] = useState(0);

  const refresh = useCallback(() => {
    setRefreshTrigger(t => t + 1);
  }, []);

  useEffect(() => {
    let cancelled = false;

    async function fetchData() {
      setLoading(true);
      setError(null);
      try {
        const [libRes, plRes] = await Promise.all([
          getLibrary().catch(() => ({ tracks: [], albums: [] })),
          getPlaylists().catch(() => ({ playlists: [] })),
        ]);

        if (!cancelled) {
          setTracks(libRes.tracks || []);
          setAlbums(libRes.albums || []);
          setPlaylists(plRes.playlists || []);
        }
      } catch (err) {
        if (!cancelled) setError(err.message);
      } finally {
        if (!cancelled) setLoading(false);
      }
    }

    fetchData();
    return () => { cancelled = true; };
  }, [refreshTrigger]);

  return { tracks, albums, playlists, loading, error, refresh };
}
