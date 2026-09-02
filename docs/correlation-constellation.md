# Correlation constellation

The correlation constellation is Lyrahcle's cross-domain investigation workspace. It turns the mission data already loaded for the authenticated operator into an interactive evidence fabric spanning satellites, SDR ground stations, RF anomalies, incidents, assets, programs and CVE exposure alerts.

## Trust model

The graph does not infer undocumented relationships. Every edge is derived from an identifier or exact contextual match that already exists in Lyrahcle data: `satellite_id`, `agent_id`, program ownership, satellite-tagged RF observations, or an exact asset-value/CVE exposure match. A CVE-to-incident link is added only when the incident text explicitly mentions that CVE identifier.

The view is therefore an investigation aid over loaded evidence, not an automatic root-cause claim and not a replacement for the existing candidate-only correlation policy. Missing permissions or page limits can produce a partial graph.

## Orbital focus

Selecting a node makes it the center of the constellation. Direct neighbors move to the first orbit, second-degree evidence to the next orbit and broader context to the outer orbit. Operators can choose one, two or three hops. This keeps dense mission context readable without discarding the underlying graph.

The time window affects event evidence such as incidents, RF anomalies, RF observations and CVE alerts. Stable mission topology remains visible so operators can distinguish a quiet period from missing infrastructure.

## Explainable routes

`Trace from here` pins the selected node as the route origin. Selecting another node computes the shortest path through the loaded evidence fabric. Every node and edge in that path is highlighted, and the context panel lists the route in order.

This can answer questions such as:

- Which ground-station evidence connects to this satellite?
- Which incident provides a path from an SDR station to a CVE?
- Which asset creates the exact exposure link between a satellite identity and a vulnerability alert?
- Which node is acting as the highest-connectivity hub in the currently loaded mission context?

If there is no path in the loaded evidence, Lyrahcle says so rather than fabricating a relationship.

## Rendering architecture

The constellation is rendered with a real WebGL scene built on three.js, `@react-three/fiber` and `@react-three/drei`. The previous in-house SVG projection has been removed. `@react-three/fiber` is pinned to the 8.x line because 9.x requires React 19 and Lyrahcle runs React 18.

Responsibilities are separated so the renderer stays replaceable:

- `src/lib/correlationGraph.ts` builds the evidence model. It is unchanged by the rebuild and still imports nothing from the renderer.
- `src/lib/correlationLayout3d.ts` turns the model into world-space coordinates. It has no three.js import, so the layout is unit tested as pure mathematics.
- `src/components/correlation/` holds the scene, the node geometry, the link and route rendering, the camera rig and the HUD.

### Layout engine

Nodes are placed on orbital shells derived from breadth-first distance to the focus node. Within a shell, nodes of the same kind share an azimuth sector, which produces visual clustering by type. Positions are then relaxed with a small deterministic force pass: attraction along loaded edges, repulsion between nodes closer than a minimum separation, and reprojection towards the shell radius.

The layout is seeded from node identifiers rather than a random source, so the same evidence always produces the same constellation. Two properties matter operationally:

- With zero links the shell distribution alone still spreads mission topology into a readable scene.
- With many links the relaxation pass pulls genuinely connected evidence together without letting nodes collapse onto each other.

### Visual language

Node geometry encodes kind without relying on colour alone: satellites are orbs with orbital rings, incidents are octahedra, CVEs are thin hexagonal plates, ground stations and SDR agents are pillars with a dish, and assets and programs are prisms. Evidence links are thin additive curves merged into a single line geometry, which keeps the draw-call count low. Only an actively traced route receives width, brightness and animation, so the explainable path is the loudest element on screen.

Labels are deliberately sparse. They render for the selected node, the hovered node, nodes on the traced route and the highest-connectivity hubs.

Because the layout groups each node kind into its own azimuth sector, the layout engine also reports a centroid per kind. Those centroids drive optional cluster markers that name each group and its size, so an operator can read the shape of the mission context before touching a single node. Markers appear only for kinds with at least two visible nodes and can be switched off from the HUD.

When a route is traced, every node outside it is dimmed and each hop carries a numbered marker matching the ordered list in the context panel, so the path can be followed in the scene and in the panel at the same time.

### Bundle cost

three.js is isolated into a `three-vendor` chunk and the scene itself is a lazy import, so neither is part of the initial document payload. Both are fetched the first time an operator opens the workspace. The chunk exceeds the shared raw-size budget and carries a documented exemption in `scripts/check-frontend-bundle.py`; the gzip budget, which governs transfer cost, is not relaxed.

## Focus mode and interaction

Selecting a node re-centres the constellation on it and eases the camera towards the new focus. Focus mode additionally dims everything outside the direct neighbourhood and any traced route, so a dense evidence fabric can be read one investigation at a time. Search offers autocomplete with arrow-key and Enter navigation and recentres the scene on the chosen node.

Controls live as a compact HUD rail with a single filter popover rather than large permanent panels, so the scene keeps most of the viewport.

## Accessibility and motion

A WebGL canvas is not keyboard navigable, so the workspace always renders a node index in the DOM alongside the scene. Every visible node is a real focusable button carrying its kind and label, and activating one selects it exactly as clicking the mesh would. The index is visually hidden on small viewports and becomes a visible rail from the extra-large breakpoint.

If the browser or environment provides no WebGL context, Lyrahcle says so and keeps the node index, search, filters, context panel and explainable routes fully usable instead of failing.

Selecting a node updates a polite live region naming the new orbital focus, its kind and its connection count, so a screen-reader operator knows the scene re-centred without being able to see it.

Continuous rotation, the route marker and node spin are disabled when the operating system requests reduced motion.

## Data and dependency boundary

The workspace adds no backend route, schema migration or graph database. It uses existing APIs and the current permission model. The rendering stack is a general-purpose 3D runtime rather than a graph engine: Lyrahcle deliberately does not depend on Cytoscape, React Flow, a D3 force simulation or another graph runtime, so the correlation semantics stay owned by Lyrahcle's own model.
