import React, { useState, useEffect } from 'react';
import { VERSION } from '../version';

export default function TitleBar({ connected }) {
  const [services, setServices] = useState({ tidal: false, qobuz: false, deezer: false });

  useEffect(() => {
    fetch('/api/health')
      .then(res => res.json())
      .then(data => {
        setServices({
          tidal: data?.services?.tidal || false,
          qobuz: data?.services?.qobuz || false,
          deezer: data?.services?.deezer || false,
        });
      })
      .catch(() => {
        setServices({ tidal: false, qobuz: false, deezer: false });
      });
  }, []);

  return (
    <div className="title-bar">
      <span className={`status-dot ${connected ? 'connected' : 'disconnected'}`} />
      <span className="logo">Sharknado <span className="version">v{VERSION}</span></span>
      <div className="service-indicators">
        {services.tidal && <span className="svc tidal on">TIDAL</span>}
        {services.qobuz && <span className="svc qobuz on">QOBUZ</span>}
        {services.deezer && <span className="svc deezer on">DEEZER</span>}
      </div>
    </div>
  );
}
