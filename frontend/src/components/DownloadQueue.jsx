import React from 'react';

const STATUS_LABELS = {
  queued: 'Queued',
  running: 'Running',
  paused: 'Paused',
  completed: 'Completed',
  failed: 'Failed',
};

export default function DownloadQueue() {
  // Placeholder — will be loaded from API + SSE
  const jobs = [];

  return (
    <div>
      <div className="view-header">
        <h2>Downloads</h2>
        <p>Manage your download queue</p>
      </div>

      {jobs.length === 0 && (
        <div className="empty-state">
          <div className="icon">⬇</div>
          <h3>No downloads yet</h3>
          <p>Submit a download URL to start downloading music.</p>
        </div>
      )}

      {jobs.length > 0 && (
        <table className="download-table">
          <thead>
            <tr>
              <th>URL</th>
              <th>Service</th>
              <th>Status</th>
              <th>Progress</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {jobs.map(job => (
              <tr key={job.id}>
                <td style={{ maxWidth: 300, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                  {job.url}
                </td>
                <td>{job.service}</td>
                <td>
                  <span className={`status-badge ${job.status}`}>
                    {STATUS_LABELS[job.status] || job.status}
                  </span>
                </td>
                <td>
                  <div className="progress-bar-container">
                    <div
                      className={`progress-bar-fill ${job.status}`}
                      style={{ width: `${Math.round(job.progress || 0)}%` }}
                    />
                  </div>
                </td>
                <td>
                  <div style={{ display: 'flex', gap: 4 }}>
                    {job.status === 'running' && (
                      <button className="btn btn-sm">⏸</button>
                    )}
                    {job.status === 'paused' && (
                      <button className="btn btn-sm">▶</button>
                    )}
                    {(job.status === 'queued' || job.status === 'paused') && (
                      <button className="btn btn-sm btn-danger">✕</button>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
