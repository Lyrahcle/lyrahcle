# Ground station operations

Ground Station Operations provides a ground-segment view over SDR agents that have trusted station coordinates configured in Lyrahcle.

## Station model

The current data model stores station coordinates on the SDR agent identity. Ground Station Operations therefore treats each configured agent as one station endpoint. The feature does not introduce a second station database model or duplicate configuration state.

A station is included only when latitude, longitude and altitude are all present and valid.

## Operational posture

Each station shows its current agent state, last contact, reported SDR hardware, integrity state and an explainable risk score. Risk evidence is scoped to the station agent identifier before it is evaluated.

The overview distinguishes online, attention and offline stations and supports filtering and search by station agent identifier.

## Current orbital visibility

Current visibility is calculated in the browser from the trusted station coordinates and the propagated satellite positions already returned by Lyrahcle. A satellite is considered visible when its calculated elevation is at least 0 degrees above the local horizon.

This is a current-geometry indicator rather than a replacement for the server-side pass prediction workflow. Operators can continue to use Orbital Operations when they need AOS, LOS, peak elevation and future pass windows.

## RF and security evidence

When the active role can read RF data, the station profile shows satellite-tagged RF observations and recent captures from the last 24 hours. It also surfaces recent RF anomalies scoped to the station agent.

When the active role can read incidents, active incidents scoped to the station agent are included in the station posture and profile.

Unavailable data domains remain permission-aware and are not fetched solely to populate the station page.
