#!/usr/bin/env python3
"""Tidal search wrapper — called by Go backend via subprocess.

Usage: tidal_search.py <query> [limit]

Outputs JSON array of normalized search results to stdout.
"""
import json
import sys
import time
from pathlib import Path

import tidalapi


def load_session(token_file: str) -> tidalapi.Session:
    """Load a tidalapi session from a token.json file."""
    session = tidalapi.Session()

    data = json.loads(Path(token_file).read_text())
    session.load_oauth_session(
        token_type=data.get("token_type", "Bearer"),
        access_token=data.get("access_token", ""),
        refresh_token=data.get("refresh_token", ""),
        expiry_time=data.get("expiry_time"),
        is_pkce=data.get("is_pkce", False),
    )
    return session


def search(session: tidalapi.Session, query: str, limit: int = 10) -> list:
    """Search Tidal and return normalized results."""
    results = session.search(query, models=[tidalapi.Track], limit=limit)

    tracks = []
    if hasattr(results, "tracks") and results.tracks:
        for t in results.tracks:
            artist_name = t.artist.name if t.artist else ""
            album_name = t.album.name if t.album else ""
            tracks.append({
                "id": f"tidal:{t.id}",
                "provider": "tidal",
                "provider_id": str(t.id),
                "title": t.name or "",
                "artist": artist_name,
                "album": album_name,
                "duration": getattr(t, "duration", 0) or 0,
                "cover_url": t.album.cover(640) if t.album and t.album.cover(640) else "",
                "type": "track",
                "quality": "hi_res" if getattr(t, "audio_quality", "") == "HI_RES_LOSSLESS" else "hifi",
            })
    return tracks


def main():
    if len(sys.argv) < 2:
        print(json.dumps({"error": "usage: tidal_search.py <query> [limit]"}))
        sys.exit(1)

    query = sys.argv[1]
    limit = int(sys.argv[2]) if len(sys.argv) > 2 else 10

    # Default token path — can be overridden via env
    token_file = sys.argv[3] if len(sys.argv) > 3 else \
        "/home/shark/.config/tidal_dl_ng/token.json"

    try:
        session = load_session(token_file)
        if not session.check_login():
            print(json.dumps({"error": "tidal login failed — token expired?"}))
            sys.exit(1)

        results = search(session, query, limit)
        print(json.dumps(results))
    except Exception as e:
        print(json.dumps({"error": str(e)}))
        sys.exit(1)


if __name__ == "__main__":
    main()
