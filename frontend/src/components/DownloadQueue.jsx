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

function getJobLabel(job) {
  if (job.title && job.title !== job.url) return job.title;
  
  if (!job.url) return 'Unknown';
  
  const url = new URL(job.url);
  const hostname = url.hostname.replace(/^www\./, '');
  
  // Try to extract service and ID from path
  const parts = url.pathname.split('/').filter(Boolean);
  
  if (hostname.includes('tidal')) {
    if (parts[0] === 'track' && parts[1]) return `TIDAL: Track ${parts[1]}`;
    if (parts[0] === 'album' && parts[1]) return `TIDAL: Album ${parts[1]}`;
    if (parts[0] === 'artist' && parts[1]) return `TIDAL: Artist ${parts[1]}`;
    if (parts[0] === 'playlist' && parts[1]) return `TIDAL: Playlist ${parts[1]}`;
    if (parts[0] === 'video' && parts[1]) return `TIDAL: Video ${parts[1]}`;
    return 'TIDAL';
  }
  
  if (hostname.includes('qobuz')) {
    if (parts[0] === 'track' && parts[1]) return `Qobuz: Track ${parts[1]}`;
    if (parts[0] === 'album' && parts[1]) return `Qobuz: Album ${parts[1]}`;
    if (parts[0] === 'artist' && parts[1]) return `Qobuz: Artist ${parts[1]}`;
    if (parts[0] === 'playlist' && parts[1]) return `Qobuz: Playlist ${parts[1]}`;
    return 'Qobuz';
  }
  
  if (hostname.includes('deezer')) {
    if (parts[0] === 'track' && parts[1]) return `Deezer: Track ${parts[1]}`;
    if (parts[0] === 'album' && parts[1]) return `Deezer: Album ${parts[1]}`;
    if (parts[0] === 'artist' && parts[1]) return `Deezer: Artist ${parts[1]}`;
    if (parts[0] === 'playlist' && parts[1]) return `Deezer: Playlist ${parts[1]}`;
    return 'Deezer';
  }
  
  if (hostname.includes('youtube') || hostname.includes('youtu.be')) return 'YouTube';
  if (hostname.includes('spotify')) return 'Spotify';
  
  // Fallback to job title or hostname
  return job.title || hostname;
}

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
                  {getJobLabel(job)}
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
