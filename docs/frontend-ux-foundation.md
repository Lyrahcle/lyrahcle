# Frontend UX foundation

Lyrahcle uses shared UI primitives for transient feedback, asynchronous state and keyboard navigation.

## Toast notifications

`ToastProvider` is mounted once around the application. Success and informational notifications are announced politely and dismiss automatically after six seconds. Warning and error notifications remain visible until the operator dismisses them. The viewport keeps at most four notifications and supports deduplication for repeated operational events.

Toasts are for transient outcomes such as a role assignment or acknowledgement. Persistent failures that affect the current workspace remain visible in the workspace itself.

## Async and empty states

`StatusState` provides five explicit states: empty, loading, error, offline and forbidden. Loading is exposed as a polite busy status. Errors use an alert. Normal empty states are not live regions.

The rate-limit workspace demonstrates the complete loading, failure, offline and retry flow. Incidents, threat intelligence, asset alerts and RBAC empty states use the same presentation primitive.

## Keyboard access

Authenticated workspaces expose a skip link to `#main-content`. Interactive controls receive a consistent visible focus outline when navigating with the keyboard.

## Accessibility rule

Live regions are reserved for dynamic information that needs announcement. Static empty states and ordinary descriptive content are not marked as live regions.
