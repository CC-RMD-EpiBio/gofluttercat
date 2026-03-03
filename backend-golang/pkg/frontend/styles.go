package frontend

const customCSS = `
:root {
	--pico-font-size: 16px;
}
body {
	min-height: 100vh;
	display: flex;
	flex-direction: column;
}
main.container {
	flex: 1;
	max-width: 720px;
	padding-top: 1rem;
}
.instrument-card {
	border: 1px solid var(--pico-muted-border-color);
	border-radius: var(--pico-border-radius);
	padding: 1rem 1.25rem;
	margin-bottom: 0.75rem;
	cursor: pointer;
	transition: border-color 0.2s;
}
.instrument-card:has(input:checked) {
	border-color: var(--pico-primary);
	background: var(--pico-primary-focus);
}
.instrument-card input[type="radio"] {
	margin-right: 0.5rem;
}
.scale-chips {
	display: flex;
	flex-wrap: wrap;
	gap: 0.25rem;
	margin-top: 0.5rem;
}
.scale-chip {
	font-size: 0.75rem;
	padding: 0.15rem 0.5rem;
	border-radius: 1rem;
	background: var(--pico-secondary-background);
	color: var(--pico-secondary-inverse);
}
.cat-settings {
	margin: 1rem 0;
}
.cat-settings summary {
	cursor: pointer;
	font-size: 0.9rem;
	color: var(--pico-muted-color);
}
.cat-settings-grid {
	display: grid;
	grid-template-columns: 1fr 120px;
	gap: 0.5rem 1rem;
	align-items: center;
	margin-top: 0.75rem;
}
.cat-settings-grid label {
	font-size: 0.85rem;
	margin: 0;
}
.cat-settings-grid input {
	margin: 0;
}
.choice-grid {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
	margin: 1rem 0;
}
.choice-btn {
	width: 100%;
	text-align: left;
	position: relative;
}
.choice-number {
	display: inline-block;
	width: 1.5rem;
	font-weight: bold;
}
.choice-key-hint {
	position: absolute;
	right: 0.5rem;
	top: 50%;
	transform: translateY(-50%);
	font-size: 0.7rem;
	padding: 0.1rem 0.4rem;
	border-radius: 3px;
	background: var(--pico-muted-border-color);
	color: var(--pico-muted-color);
	font-family: monospace;
	line-height: 1;
}
.skip-btn {
	margin-top: 0.5rem;
	position: relative;
}
.score-card {
	border: 1px solid var(--pico-muted-border-color);
	border-radius: var(--pico-border-radius);
	padding: 1rem 1.25rem;
	margin-bottom: 1rem;
}
.score-card h3 {
	margin-bottom: 0.75rem;
}
.score-dist-section {
	margin-bottom: 0.75rem;
}
.score-dist-label {
	font-size: 0.85rem;
	margin: 0 0 0.25rem 0;
	padding-left: 0.5rem;
	border-left: 3px solid;
}
.dist-primary {
	border-color: var(--pico-primary);
	color: var(--pico-primary);
}
.dist-secondary {
	border-color: var(--pico-secondary);
	color: var(--pico-secondary);
}
.score-stats {
	display: flex;
	gap: 2rem;
	margin-bottom: 0.5rem;
}
.score-stat-label {
	font-size: 0.8rem;
	color: var(--pico-muted-color);
}
.score-stat-value {
	font-size: 1.25rem;
	font-weight: bold;
}
.density-plot {
	margin: 0.5rem 0 0;
}
.density-plot svg {
	width: 100%;
	height: auto;
}
.forest-plot {
	margin: 1.5rem 0;
}
.forest-plot svg {
	width: 100%;
	height: auto;
}
.results-legend {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	margin-bottom: 1rem;
	font-size: 0.85rem;
	color: var(--pico-muted-color);
}
.legend-swatch {
	display: inline-block;
	width: 14px;
	height: 14px;
	border-radius: 3px;
}
.legend-primary {
	background: var(--pico-primary);
}
.legend-secondary {
	background: var(--pico-secondary);
	margin-left: 1rem;
}
.htmx-indicator {
	display: none;
}
.htmx-request .htmx-indicator {
	display: inline-block;
}
.htmx-request.htmx-indicator {
	display: inline-block;
}
.question-area {
	min-height: 200px;
}
`
