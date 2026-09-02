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

## Accessibility and motion

SVG nodes are keyboard focusable and activatable with Enter or Space. Search provides a conventional button-based way to jump to nodes. Critical and selected nodes use visual halos without relying on color alone. Flow and pulse animations are disabled when the operating system requests reduced motion.

## Data and dependency boundary

The workspace adds no backend route, schema migration or graph database. It uses existing APIs and the current permission model. The rendering and layout engine is implemented inside Lyrahcle rather than adding Cytoscape, React Flow, D3 force simulation or another graph runtime dependency.


## Minimal 3D view

The constellation uses a lightweight mathematical perspective projection rather than a graph or WebGL dependency. Nodes occupy three-dimensional orbital planes and are depth-sorted before projection. Operators can orbit the camera by dragging the scene, while zoom and recenter controls remain available. The visual treatment is deliberately minimal: thin orbital guides, restrained depth cues, subtle node shadows, and emphasis reserved for selected evidence and explainable routes.

The 3D position changes presentation only. It does not infer undocumented relationships or alter the loaded evidence.

## Minimal 3D presentation

The flagship view uses a restrained 3D projection with pointer-driven camera orbit, depth ordering, perspective scaling, thin orbital guides, and curved evidence links. The visual layer stays intentionally minimal: no decorative graph engine, no continuous node pulsing, and no inferred relationships. Only an actively traced explainable route receives animated emphasis.

## Viewport-first 3D workspace

The constellation canvas is the workspace, not a secondary chart. Controls, metrics, search and selected-node context are compact overlays around the 3D scene. When mission topology contains nodes but no current links, Lyrahcle keeps up to 30 authorized nodes visible as spatial context instead of collapsing to one isolated focus.
