import React, { useState, useEffect, useCallback } from 'react';
import { getDownloads, pauseDownload, cancelDownload } from '../api/client';
import { useSSE } from '../hooks/useSSE';

const STATUS_LABELS = {
  queued: 'Queued',
  running: 'Running',
  paused: 'Paused',
  completed: 'Completed',
  failed: 'Failed',
  cancelled: 'Cancelled',
};

export default function DownloadQueue({ refreshTrigger }) {
  const [jobs, setJobs] = useState([]);
  const [loading, setLoading] = useState(true);
  const { addEventListener } = useSSE('/api/events');

  const fetchJobs = useCallback(async () => {
    try {
      const data = await getDownloads();
      setJobs(data.jobs || []);
    } catch (err) {
      console.error('Failed to fetch downloads:', err);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchJobs();
  }, [fetchJobs, refreshTrigger]);

  // Subscribe to SSE events for real-time updates
  useEffect(() => {
    const cleanup1 = addEventListener('job.updated', () => {
      fetchJobs();
    });
    const cleanup2 = addEventListener('job.log', () => {
      fetchJobs();
    });
    return () => {
      cleanup1();
      cleanup2();
    };
  }, [addEventListener, fetchJobs]);

  const handlePause = async (id) => {
    try {
      await pauseDownload(id);
      fetchJobs();
    } catch (err) {
      console.error('Pause failed:', err);
    }
  };

  const handleCancel = async (id) => {
    try {
      await cancelDownload(id);
      fetchJobs();
    } catch (err) {
      console.error('Cancel failed:', err);
    }
  };

  return (
    <div>
      <div className="view-header">
        <h2>Downloads</h2>
        <p>Manage your download queue</p>
      </div>

      {loading && (
        <div className="empty-state">
          <span className="spinner spinner-lg" />
        </div>
      )}

      {!loading && jobs.length === 0 && (
        <div className="empty-state">
          <div className="icon">&#11015;</div>
          <h3>No downloads yet</h3>
          <p>Submit a download URL or click the download button on search results.</p>
        </div>
      )}

      {!loading && jobs.length > 0 && (
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
                      <button className="btn btn-sm" onClick={() => handlePause(job.id)} title="Pause">&#9646;&#9646;</button>
                    )}
                    {(job.status === 'queued' || job.status === 'paused') && (
                      <button className="btn btn-sm btn-danger" onClick={() => handleCancel(job.id)} title="Cancel">&#10005;</button>
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
