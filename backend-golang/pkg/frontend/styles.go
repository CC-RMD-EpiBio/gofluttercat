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
.choice-grid {
	display: flex;
	flex-direction: column;
	gap: 0.5rem;
	margin: 1rem 0;
}
.choice-btn {
	width: 100%;
	text-align: left;
}
.choice-number {
	display: inline-block;
	width: 1.5rem;
	font-weight: bold;
}
.skip-btn {
	margin-top: 0.5rem;
}
.score-card {
	border: 1px solid var(--pico-muted-border-color);
	border-radius: var(--pico-border-radius);
	padding: 1rem 1.25rem;
	margin-bottom: 1rem;
}
.score-card h3 {
	margin-bottom: 0.5rem;
}
.score-stats {
	display: flex;
	gap: 2rem;
	margin-bottom: 0.75rem;
}
.score-stat-label {
	font-size: 0.8rem;
	color: var(--pico-muted-color);
}
.score-stat-value {
	font-size: 1.25rem;
	font-weight: bold;
}
.forest-plot svg {
	width: 100%;
	height: auto;
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
