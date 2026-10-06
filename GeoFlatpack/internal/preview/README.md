# Terminal map preview

This package runs a local, embedded MapLibre GL JS page in a persistent headless
Chromium browser. The TUI uploads its PNG screenshots through Kitty graphics
virtual placements and draws Unicode placeholders inside the Preview pane.
It has no dependency on the GitHub Pages demo in `web/`.

Runtime requirements are a terminal supporting Kitty graphics **and** virtual
placements, plus an installed Chrome, Chromium, or Brave browser with WebGL.
`GFP_PREVIEW_BROWSER` can point to another Chromium executable. Unsupported
terminals keep the editor usable and display an explanatory message. The browser
is only started after both terminal capability probes succeed.

Preview follows the highlighted group while browsing Features, without opening
its editor or changing completion status. Styling and Controls show the confirmed
editing group. Layers and Category retain that confirmed group, or use the active
category's highlighted feature if none has been chosen. Category selection remains
explicit: before activation, the pane displays `Choose a category to preview`.
An editor without a chosen group displays `Choose a feature to preview`.
The first usable matching feature in loaded source
order represents the group, including all parts of multipart geometry. Only that
feature and its SVG vertex companions are sent to the renderer with its full stack.
Snapshots use the export generator, including embedded SVGs and stored vertex
companions. Unconfigured groups use the generator's defaults. Pending invalid
inputs keep their last valid value or omit a property that has no valid value.
Geometry is reprojected independently to WGS84. Lines and polygons fit tightly
with a maximum zoom of 22; points use a maximum zoom of 16. Circles, strokes, and
SVGs keep their configured pixel sizes. Geometry is cached across style edits.

Press D outside input editing to toggle compact sample geometry for all previews
in the current session. The header shows `Preview · Sample`. Samples stay near
the real representative's centre: one point, a regular hexagon 100 metres across
opposite vertices, or a five-vertex line 150 metres wide with alternating vertical
offsets of ±25 metres. Samples keep the original properties, full style stack,
and pixel sizes. SVGs render at the single point, six distinct hexagon vertices,
or five line vertices. Sample geometry never changes loaded data or exports.

Each visited feature target and geometry mode retains its latest successfully
uploaded image for the current session. Returning with unchanged styling restores its image and
warning immediately, without another render or upload. Valid live edits replace
that target's cached image after a successful upload; stale or failed renders do
not replace it. Resizing the preview or changing terminal cell dimensions clears
the image cache. Help keeps the cache, and exit releases all cached images.

The OpenFreeMap Liberty basemap needs network access. If it cannot load, the
styled features remain on a neutral background and the pane header reports
`Basemap unavailable`. Navigation changes debounce for 150 ms; each render has
a 10-second deadline. New requests cancel old ones and discard stale results.
Help, tiny panes, writing, and exit suspend rendering. Exit closes the browser,
its temporary profile, and the local server and releases only app-owned images.

## Rebuild the embedded renderer

The bundled files in `assets/` are checked in, so normal Go builds and runtime
do not need Node. After changing `frontend/`, rebuild from that directory:

```sh
npm ci
npm test
npm run build
```

The renderer and worker are bundled locally; only basemap resources use external
URLs. Keep the generated assets in the same change as their frontend sources.

## Verification

From the Go module root:

```sh
go test ./...
go vet ./...
GFP_PREVIEW_BROWSER_TEST=1 go test -v ./internal/preview -run TestBrowser -count=1
```

The optional integration test launches an installed browser and uses local
basemap fixtures. It checks single-feature framing and symbol sizes, actual
polygon pixels, SVG vertex icons, offline
fallback, cancellation, browser reuse, screenshot dimensions, and server cleanup.
Set `GFP_PREVIEW_TEST_PNG` to save the resulting screenshot for inspection.
