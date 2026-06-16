import { useState, useEffect, useCallback, useRef } from 'react';

export function useSSE(url) {
  const [connected, setConnected] = useState(false);
  const [lastEvent, setLastEvent] = useState(null);
  const [events, setEvents] = useState({});
  const eventSourceRef = useRef(null);

  useEffect(() => {
    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.onopen = () => setConnected(true);

    es.onerror = () => {
      setConnected(false);
    };

    es.addEventListener('connected', (e) => {
      setConnected(true);
      try {
        const data = JSON.parse(e.data);
        setLastEvent({ type: 'connected', data });
      } catch { /* ignore parse errors */ }
    });

    es.onmessage = (e) => {
      try {
        const data = JSON.parse(e.data);
        setLastEvent({ type: 'message', data });
      } catch { /* ignore parse errors */ }
    };

    return () => {
      es.close();
      eventSourceRef.current = null;
      setConnected(false);
    };
  }, [url]);

  const addEventListener = useCallback((eventType, handler) => {
    if (eventSourceRef.current) {
      eventSourceRef.current.addEventListener(eventType, handler);
    }
  }, []);

  return { events, connected, lastEvent, addEventListener };
}
