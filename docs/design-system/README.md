# Design system

<!-- human -->
pile looks like the Sorter: its design system, copied as it is.
<!-- /human -->

This app follows the Sorter design system, `software/sorter-design-system`
in `basicallysource/sorter-v2`: its rules and docs are there. Its tokens
are copied into `app/src/app.css` and its components into
`app/src/lib/components/`, unchanged. A component or token this app needs
is added there first and copied here again; never fork one in this repo.
The app's own pieces (built from those components) are
`app/src/lib/pile/`. `/design` in the app shows the components this app
uses on sample data.
