# Frontend TypeScript migration

Lyrahcle uses a gradual TypeScript migration. `allowJs` remains enabled so each component can move independently while the local and CI gates run `npm run typecheck` before tests and the production build.

## Current phase

The shared frontend library layer is already TypeScript. Block 14 moves the application boundary and selected security surfaces to typed files:

- `App.tsx`
- `components/Dashboard.tsx`
- `hooks/useHashRoute.ts`
- `components/dashboard/navigation.ts`
- `components/Panel.tsx`
- `components/RBACPanel.tsx`
- `components/RateLimitPolicyPanel.tsx`
- `components/SecretProviderPanel.tsx`
- `components/IncidentResponsePanel.tsx`
- shared RBAC types in `types/security.ts`

Hash navigation is no longer owned by `App`; the typed route hook owns URL synchronization and returns a `RouteTab`-checked navigation callback. Permission-driven dashboard destinations are built by a typed navigation model instead of an inline untyped array.

RBAC and platform-security panels now type their props, remote state, form events and error handling. The general `Panel` component also has typed optional props, which prevents JavaScript inference from making optional presentation fields accidentally required when consumed from TSX.

## Migration rules

New or substantially refactored frontend files should use `.ts` or `.tsx`. Existing `.jsx` files can remain until touched. A migration should preserve UI copy, permissions and lazy-loading behavior unless the block explicitly changes product behavior.

Avoid converting a component by extension alone. Prefer typed props, typed state at API boundaries and `unknown` error handling. Temporary broad record types are acceptable at legacy API boundaries while response schemas remain untyped, but security-sensitive components should use explicit domain interfaces. Until a JSX child is migrated, TSX parents must mark that interop boundary explicitly instead of accepting inferred JavaScript default-prop types such as `never[]` or zero-argument callbacks.

## Next candidates

The next high-value candidates are `SigningKeysPanel`, `UsersPanel`, `IncidentsPanel`, `RFHeatMap` and `OrbitalOperationsPanel`. `Dashboard.tsx` should continue to lose responsibilities as those workspaces gain typed data hooks or section components.
