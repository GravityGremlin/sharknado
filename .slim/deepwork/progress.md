# Sharknado Deepwork Session - Major Layout Redesign

## Session Date: 2026-06-17

## Current Status
- Version: 1.0.2 deployed
- Backend APIs: ✅ Complete
- Playlist/Queue: ✅ Working
- Layout: ❌ Backwards - search in sidebar, main area empty

## Problem Statement
The layout is fundamentally inverted:
- Sidebar (220px) contains: Search bar, source filters, results (cramped)
- Main area (80% of screen): Completely empty
- Player bar at bottom: Full width, correct

## Target Layout
```
[Sidebar 200px - Nav Only] | [Main Area - Search + Results Grid]
  BROWSE                   |  [Search Input Full Width] [TIDAL] [QOBUZ]
    Search                 |  ----------------------------------------
    Library                |  Results: Album/Track cards in grid
    Downloads              |
    Queue (0)              |
                           |
  PLAYLISTS                |
    + New Playlist         |
                           |
[Player Bar - Full Width, Pinned Bottom]
```

## Phased Implementation Plan

### Phase 1: Restructure App Layout & Sidebar
- Move SearchView out of sidebar into Main area
- Sidebar = pure navigation (Browse, Library, Downloads, Queue, Playlists)
- Update App.jsx routing to render SearchView in main area
- Sidebar width: 200px fixed

### Phase 2: Search Bar in Main Area
- SearchView becomes header of main content area
- Search input full width
- Source toggles as subtle pills near search
- Search button secondary

### Phase 3: Results Grid Layout
- Replace grouped list with responsive card grid
- Album art cards with track info overlay
- Hover states for play/download actions
- Empty state: featured/recent content

### Phase 4: Typography & Visual Hierarchy
- Stronger section labels in sidebar
- Better weight differentiation
- Subtle source toggle pills
- Consistent spacing system

### Phase 5: Responsive & Polish
- Mobile breakpoint adjustments
- Card grid responsiveness
- Performance: CSS containment, lazy loading

## Oracle Review Required
- [ ] Phase 1 plan review
- [ ] Phase 2-3 implementation review
- [ ] Phase 4-5 review

## Build/Deploy
- [ ] Build verification
- [ ] Docker rebuild
- [ ] Deploy to shark.gravitywell.xyz