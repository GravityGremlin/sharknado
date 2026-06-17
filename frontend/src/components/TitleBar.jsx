import React from 'react';
import { VERSION } from '../version';

export default function TitleBar({ connected }) {
  return (
    <div className="title-bar">
      <span className={`status-dot ${connected ? 'connected' : 'disconnected'}`} />
      <span className="logo">Sharknado <span className="version">v{VERSION}</span></span>
      <div className="service-indicators">
        <span className="svc tidal on">TIDAL</span>
        <span className="svc qobuz on">QOBUZ</span>
      </div>
    </div>
  );
}
