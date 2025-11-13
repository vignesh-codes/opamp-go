package view

// GetTopologyViewHTML returns the HTML for the topology visualization view
func GetTopologyViewHTML() string {
	return `<!DOCTYPE html>
<html>
<head>
	<title>OpAMP Topology Visualization</title>
	<style>
		body {
			font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
			margin: 0;
			padding: 16px;
			background: #f9fafb;
		}
		.container {
			max-width: 100%;
			background: white;
			padding: 16px;
			border-radius: 6px;
			box-shadow: 0 1px 3px rgba(0,0,0,0.1);
		}
		h1 {
			color: #111827;
			margin-top: 0;
			font-size: 20px;
			font-weight: 600;
			margin-bottom: 12px;
		}
		#graph {
			width: 100%;
			height: 600px;
			border: 1px solid #e5e7eb;
			background: #ffffff;
			position: relative;
			overflow: auto;
			min-height: 500px;
		}
		.node {
			position: absolute;
			padding: 8px 12px;
			border-radius: 6px;
			font-size: 12px;
			font-weight: 500;
			cursor: move;
			box-shadow: 0 2px 4px rgba(0,0,0,0.1);
			transition: all 0.2s ease;
			z-index: 10;
			min-width: 100px;
			max-width: 140px;
			text-align: center;
			white-space: normal;
			overflow: hidden;
			word-wrap: break-word;
			user-select: none;
			border: 1.5px solid rgba(255,255,255,0.3);
		}
		.node:hover {
			z-index: 20;
			box-shadow: 0 4px 8px rgba(0,0,0,0.15);
			transform: translateY(-1px);
			border-color: rgba(255,255,255,0.5);
		}
		.node.dragging {
			opacity: 0.85;
			cursor: grabbing;
			z-index: 100;
			box-shadow: 0 6px 12px rgba(0,0,0,0.25);
		}
		.node.agent {
			background: #6366f1;
			color: white;
			border-color: rgba(255,255,255,0.4);
		}
		.node.receiver {
			background: #3b82f6;
			color: white;
			border-color: rgba(255,255,255,0.4);
		}
		.node.processor {
			background: #f59e0b;
			color: white;
			border-color: rgba(255,255,255,0.4);
		}
		.node.exporter {
			background: #8b5cf6;
			color: white;
			border-color: rgba(255,255,255,0.4);
		}
		.node.endpoint {
			background: #ef4444;
			color: white;
			border-color: rgba(255,255,255,0.4);
		}
		.node-label {
			display: inline-block;
			font-size: 9px;
			margin-top: 3px;
			opacity: 0.9;
			font-weight: 500;
			letter-spacing: 0.3px;
			text-transform: uppercase;
			padding: 1px 4px;
			background: rgba(255,255,255,0.15);
			border-radius: 3px;
		}
		.rate-label {
			display: block;
			font-size: 10px;
			margin-top: 4px;
			font-weight: 600;
			line-height: 1.3;
			padding: 3px 0;
			border-top: 1px solid rgba(255,255,255,0.15);
			margin-top: 4px;
			padding-top: 4px;
		}
		svg {
			position: absolute;
			top: 0;
			left: 0;
			width: 100%;
			height: 100%;
			pointer-events: none;
			z-index: 1;
		}
		line {
			stroke: #9ca3af;
			stroke-width: 1.5;
			marker-end: url(#arrowhead);
			opacity: 0.6;
			transition: all 0.2s ease;
		}
		line:hover {
			opacity: 0.9;
			stroke-width: 2;
		}
		.controls {
			margin-bottom: 15px;
		}
		button {
			background: #6366f1;
			color: white;
			border: none;
			padding: 6px 14px;
			border-radius: 4px;
			cursor: pointer;
			margin-right: 8px;
			font-weight: 500;
			font-size: 12px;
			transition: all 0.2s ease;
		}
		button:hover {
			background: #4f46e5;
		}
		button:active {
			background: #4338ca;
		}
		.info {
			margin-top: 12px;
			padding: 10px 14px;
			background: #f3f4f6;
			border-radius: 4px;
			font-size: 12px;
			border-left: 3px solid #3b82f6;
			color: #374151;
		}
	</style>
</head>
<body>
	<div class="container">
		<h1>OpAMP Topology Graph</h1>
		<div class="controls">
			<button onclick="loadTopology()">Refresh</button>
			<button onclick="toggleAutoRefresh()" id="autoRefreshBtn">Auto-Refresh: OFF</button>
			<button onclick="resetLayout()">Reset Layout</button>
			<button onclick="clearManualPositions()">Clear Manual Positions</button>
			<span style="margin-left: 16px; color: #6b7280; font-size: 12px;">💡 Drag nodes to reposition</span>
			<span style="margin-left: 16px; color: #10b981; font-size: 12px; font-weight: 500;">⚡ Real-time rates</span>
			<div style="margin-top: 8px; display: flex; gap: 16px; align-items: center;">
				<span style="font-size: 11px; font-weight: 500; color: #6b7280;">Pipeline Types:</span>
				<span id="pipelineLegend" style="font-size: 11px;"></span>
			</div>
		</div>
		<div id="graph">
			<svg id="edges">
				<defs>
					<marker id="arrowhead" markerWidth="8" markerHeight="8" refX="7" refY="2.5" orient="auto">
						<polygon points="0 0, 8 2.5, 0 5" fill="#9ca3af" />
					</marker>
				</defs>
			</svg>
		</div>
		<div class="info" id="info">Loading topology...</div>
	</div>

	<script>
		let nodePositions = {};
		let nodes = [];
		let edges = [];
		let draggedNode = null;
		let dragOffset = { x: 0, y: 0 };
		let isDragging = false;
		let autoRefreshInterval = null;
		let autoRefreshEnabled = false;

		function toggleAutoRefresh() {
			autoRefreshEnabled = !autoRefreshEnabled;
			const btn = document.getElementById('autoRefreshBtn');
			if (autoRefreshEnabled) {
				btn.textContent = 'Auto-Refresh: ON (5s)';
				btn.style.background = '#45a049';
				autoRefreshInterval = setInterval(loadTopology, 5000);
			} else {
				btn.textContent = 'Auto-Refresh: OFF';
				btn.style.background = '#4CAF50';
				if (autoRefreshInterval) {
					clearInterval(autoRefreshInterval);
					autoRefreshInterval = null;
				}
			}
		}

		function loadTopology() {
			const infoEl = document.getElementById('info');
			if (infoEl) {
				infoEl.textContent = 'Loading topology...';
			}
			
			fetch('/api/topology')
				.then(res => {
					if (!res.ok) {
						throw new Error('HTTP ' + res.status + ': ' + res.statusText);
					}
					return res.json();
				})
				.then(data => {
					nodes = data.nodes || [];
					edges = data.edges || [];
					
					if (nodes.length === 0) {
						if (infoEl) {
							infoEl.textContent = 'No nodes found. Make sure agents are connected.';
						}
						return;
					}
					
					// Update pipeline type legend dynamically
					const pipelineTypes = new Set();
					nodes.forEach(n => {
						if (n.pipeline_type) {
							pipelineTypes.add(n.pipeline_type);
						}
					});
					edges.forEach(e => {
						if (e.pipeline_type) {
							pipelineTypes.add(e.pipeline_type);
						}
					});
					
					const legendEl = document.getElementById('pipelineLegend');
					if (legendEl) {
					const pipelineColors = {
						'traces': '#3b82f6',
						'logs': '#f59e0b',
						'metrics': '#10b981'
					};
						const pipelineLabels = {
							'traces': 'Traces',
							'logs': 'Logs',
							'metrics': 'Metrics'
						};
						
						const legendItems = Array.from(pipelineTypes).sort().map(pt => {
							const color = pipelineColors[pt] || '#666';
							const label = pipelineLabels[pt] || pt;
							return '<span style="display: inline-flex; align-items: center; margin-right: 15px;"><span style="display: inline-block; width: 16px; height: 16px; background: ' + color + '; border-radius: 3px; margin-right: 5px;"></span><span>' + label + '</span></span>';
						});
						legendEl.innerHTML = legendItems.join('');
					}
					
					renderGraph();
					
					const nodesWithRates = nodes.filter(n => {
						if (!n.data_transfer) return false;
						return (n.data_transfer.spans_received_rate > 0 || n.data_transfer.spans_sent_rate > 0 ||
						        n.data_transfer.logs_received_rate > 0 || n.data_transfer.logs_sent_rate > 0 ||
						        n.data_transfer.metrics_received_rate > 0 || n.data_transfer.metrics_sent_rate > 0);
					}).length;
					
					const edgesWithRates = edges.filter(e => {
						if (!e.data_transfer) return false;
						return (e.data_transfer.spans_transferred_rate > 0 || 
						        e.data_transfer.logs_transferred_rate > 0 ||
						        e.data_transfer.metrics_transferred_rate > 0);
					}).length;
					
					let infoText = 'Found ' + nodes.length + ' nodes and ' + edges.length + ' edges';
					if (pipelineTypes.size > 0) {
						infoText += ' | Pipeline types: ' + Array.from(pipelineTypes).sort().join(', ');
					}
					if (nodesWithRates > 0 || edgesWithRates > 0) {
						infoText += ' | ⚡ Real-time rates: ' + nodesWithRates + ' nodes, ' + edgesWithRates + ' edges';
					}
					if (infoEl) {
						infoEl.textContent = infoText;
					}
				})
				.catch(err => {
					const errorMsg = 'Error loading topology: ' + err.message;
					console.error(errorMsg, err);
					if (infoEl) {
						infoEl.textContent = errorMsg;
					}
				});
		}

		function resetLayout() {
			nodePositions = {};
			localStorage.removeItem('topologyNodePositions');
			if (nodes.length > 0) {
				renderGraph();
			}
		}

		function clearManualPositions() {
			if (confirm('Clear all manually positioned nodes and reset to auto-layout?')) {
				resetLayout();
			}
		}

		function savePositions() {
			localStorage.setItem('topologyNodePositions', JSON.stringify(nodePositions));
		}

		function loadSavedPositions() {
			const saved = localStorage.getItem('topologyNodePositions');
			if (saved) {
				try {
					const parsed = JSON.parse(saved);
					Object.assign(nodePositions, parsed);
				} catch (e) {
					console.error('Failed to load saved positions:', e);
				}
			}
		}

		function calculateLayout() {
			const graph = document.getElementById('graph');
			if (!graph) return;
			
			if (graph.offsetWidth === 0) {
				graph.style.width = '100%';
			}
			if (graph.offsetHeight === 0) {
				graph.style.height = '800px';
			}
			const width = Math.max(graph.offsetWidth || 1200, 1200);
			const height = Math.max(graph.offsetHeight || 800, 800);
			
			const agents = nodes.filter(n => n.type === 'agent');
			const components = nodes.filter(n => n.type !== 'agent' && n.type !== 'endpoint');
			const endpoints = nodes.filter(n => n.type === 'endpoint');

			const layerSpacing = 100;
			const nodeSpacing = 120;
			const horizontalPadding = 80;
			const verticalPadding = 60;
			const agentY = verticalPadding;
			
			if (agents.length > 0) {
				const agentTotalWidth = Math.min(agents.length * nodeSpacing, width - 2 * horizontalPadding);
				const agentStartX = (width - agentTotalWidth) / 2 + nodeSpacing / 2;
				agents.forEach((agent, i) => {
					if (!nodePositions[agent.id]) {
						nodePositions[agent.id] = {
							x: agentStartX + i * nodeSpacing,
							y: agentY
						};
					}
				});
			}

			let maxY = agentY + layerSpacing;
			
			agents.forEach((agent, agentIdx) => {
				const agentComponents = components.filter(c => c.agent_id === agent.agent_id);
				
				const agentComponentsByTypeAndPipeline = {
					receiver: {
						traces: agentComponents.filter(c => c.type === 'receiver' && c.pipeline_type === 'traces'),
						logs: agentComponents.filter(c => c.type === 'receiver' && c.pipeline_type === 'logs'),
						metrics: agentComponents.filter(c => c.type === 'receiver' && c.pipeline_type === 'metrics'),
						other: agentComponents.filter(c => c.type === 'receiver' && !c.pipeline_type)
					},
					processor: {
						traces: agentComponents.filter(c => c.type === 'processor' && c.pipeline_type === 'traces'),
						logs: agentComponents.filter(c => c.type === 'processor' && c.pipeline_type === 'logs'),
						metrics: agentComponents.filter(c => c.type === 'processor' && c.pipeline_type === 'metrics'),
						other: agentComponents.filter(c => c.type === 'processor' && !c.pipeline_type)
					},
					exporter: {
						traces: agentComponents.filter(c => c.type === 'exporter' && c.pipeline_type === 'traces'),
						logs: agentComponents.filter(c => c.type === 'exporter' && c.pipeline_type === 'logs'),
						metrics: agentComponents.filter(c => c.type === 'exporter' && c.pipeline_type === 'metrics'),
						other: agentComponents.filter(c => c.type === 'exporter' && !c.pipeline_type)
					}
				};

				let agentPos = nodePositions[agent.id];
				if (!agentPos) {
					nodePositions[agent.id] = {
						x: horizontalPadding + agentIdx * nodeSpacing,
						y: agentY
					};
					agentPos = nodePositions[agent.id];
				}

				let currentY = agentY + layerSpacing;
				const componentWidth = 110;
				const maxComponentsPerRow = 4;
				const pipelineSpacing = 25;

				function positionComponentsByPipeline(componentsByPipeline, currentY) {
					const pipelineOrder = ['traces', 'logs', 'metrics', 'other'];
					let y = currentY;
					
					for (const pipelineType of pipelineOrder) {
						const comps = componentsByPipeline[pipelineType];
						if (comps.length === 0) continue;
						
						const rows = Math.ceil(comps.length / maxComponentsPerRow);
						const rowWidth = Math.min(comps.length, maxComponentsPerRow) * componentWidth;
						const startX = agentPos.x - rowWidth / 2 + componentWidth / 2;
						
						comps.forEach((comp, i) => {
							if (!nodePositions[comp.id]) {
								const row = Math.floor(i / maxComponentsPerRow);
								const col = i % maxComponentsPerRow;
								const compY = y + row * 70;
								nodePositions[comp.id] = {
									x: startX + col * componentWidth,
									y: compY
								};
								maxY = Math.max(maxY, compY + 40);
							} else {
								maxY = Math.max(maxY, nodePositions[comp.id].y + 40);
							}
						});
						
						y += rows * 70 + pipelineSpacing;
					}
					
					return y;
				}

				currentY = positionComponentsByPipeline(agentComponentsByTypeAndPipeline.receiver, currentY);
				currentY += 15;
				currentY = positionComponentsByPipeline(agentComponentsByTypeAndPipeline.processor, currentY);
				currentY += 15;
				positionComponentsByPipeline(agentComponentsByTypeAndPipeline.exporter, currentY);
			});

			if (endpoints.length > 0) {
				const endpointY = Math.max(height - verticalPadding, maxY + layerSpacing);
				const endpointTotalWidth = Math.min(endpoints.length * nodeSpacing, width - 2 * horizontalPadding);
				const endpointStartX = (width - endpointTotalWidth) / 2 + nodeSpacing / 2;
				
				endpoints.forEach((endpoint, i) => {
					if (!nodePositions[endpoint.id]) {
						nodePositions[endpoint.id] = {
							x: endpointStartX + i * nodeSpacing,
							y: endpointY
						};
					} else {
						maxY = Math.max(maxY, nodePositions[endpoint.id].y + 40);
					}
				});
			}
		}

		function formatCount(count) {
			if (!count || count === 0) return '0';
			if (count >= 1000000000) {
				return (count / 1000000000).toFixed(2) + 'B';
			}
			if (count >= 1000000) {
				return (count / 1000000).toFixed(2) + 'M';
			}
			if (count >= 1000) {
				return (count / 1000).toFixed(2) + 'K';
			}
			return count.toLocaleString();
		}

		function formatRate(rate) {
			if (!rate || rate === 0) return null;
			if (rate >= 1000) {
				return rate.toFixed(1) + '/s';
			}
			return rate.toFixed(2) + '/s';
		}

		function formatNodeDataTransfer(dt, pipelineType) {
			if (!dt) return '';
			let parts = [];
			const rateParts = [];
			
			// For agent nodes (pipelineType is null), show all pipeline types that have data
			// For component nodes (pipelineType is set), only show the relevant pipeline type
			if (!pipelineType) {
				// Agent node: show all pipeline types that have data or are configured
				const logsReceivedRate = dt.logs_received_rate !== undefined ? dt.logs_received_rate : 0;
				const logsSentRate = dt.logs_sent_rate !== undefined ? dt.logs_sent_rate : 0;
				// Always show logs rates if logs pipeline is configured (even if 0)
				// Check if logs pipeline exists by checking if we have logs receivers/processors/exporters
				// For now, show logs rates if the field exists (which it will if logs pipeline is configured)
				if (dt.logs_received_rate !== undefined || dt.logs_sent_rate !== undefined) {
					if (logsReceivedRate > 0 || logsSentRate > 0) {
						if (logsReceivedRate > 0 && logsSentRate > 0) {
							rateParts.push('Logs: ' + formatRate(logsReceivedRate) + ' --> ' + formatRate(logsSentRate));
						} else {
							rateParts.push('Logs: ' + formatRate(logsReceivedRate || logsSentRate));
						}
					} else {
						// Show 0 rates to indicate logs pipeline exists but no data flowing
						rateParts.push('Logs: 0/s --> 0/s');
					}
				}
				if (dt.metrics_received_rate > 0 || dt.metrics_sent_rate > 0) {
					if (dt.metrics_received_rate > 0 && dt.metrics_sent_rate > 0) {
						rateParts.push('Metrics: ' + formatRate(dt.metrics_received_rate) + ' --> ' + formatRate(dt.metrics_sent_rate));
					} else {
						rateParts.push('Metrics: ' + formatRate(dt.metrics_received_rate || dt.metrics_sent_rate));
					}
				}
				if (dt.spans_received_rate > 0 || dt.spans_sent_rate > 0) {
					if (dt.spans_received_rate > 0 && dt.spans_sent_rate > 0) {
						rateParts.push('Spans: ' + formatRate(dt.spans_received_rate) + ' --> ' + formatRate(dt.spans_sent_rate));
					} else {
						rateParts.push('Spans: ' + formatRate(dt.spans_received_rate || dt.spans_sent_rate));
					}
				}
			} else {
				// Component node: show only the relevant pipeline type
				if (pipelineType === 'traces') {
					if (dt.spans_received_rate > 0 || dt.spans_sent_rate > 0) {
						if (dt.spans_received_rate > 0 && dt.spans_sent_rate > 0) {
							rateParts.push('Spans: ' + formatRate(dt.spans_received_rate) + ' --> ' + formatRate(dt.spans_sent_rate));
						} else {
							rateParts.push('Spans: ' + formatRate(dt.spans_received_rate || dt.spans_sent_rate));
						}
					}
				}
				if (pipelineType === 'logs') {
					// Always show logs rates to indicate logs pipeline exists
					const logsReceivedRate = dt.logs_received_rate !== undefined ? dt.logs_received_rate : 0;
					const logsSentRate = dt.logs_sent_rate !== undefined ? dt.logs_sent_rate : 0;
					if (logsReceivedRate > 0 || logsSentRate > 0) {
						if (logsReceivedRate > 0 && logsSentRate > 0) {
							rateParts.push('Logs: ' + formatRate(logsReceivedRate) + ' --> ' + formatRate(logsSentRate));
						} else {
							rateParts.push('Logs: ' + formatRate(logsReceivedRate || logsSentRate));
						}
					} else {
						// Show 0 rates to indicate logs pipeline exists but no data flowing
						rateParts.push('Logs: 0/s --> 0/s');
					}
				}
				if (pipelineType === 'metrics') {
					if (dt.metrics_received_rate > 0 || dt.metrics_sent_rate > 0) {
						if (dt.metrics_received_rate > 0 && dt.metrics_sent_rate > 0) {
							rateParts.push('Metrics: ' + formatRate(dt.metrics_received_rate) + ' --> ' + formatRate(dt.metrics_sent_rate));
						} else {
							rateParts.push('Metrics: ' + formatRate(dt.metrics_received_rate || dt.metrics_sent_rate));
						}
					}
				}
			}
			if (rateParts.length > 0) {
				parts.push('⚡ ' + rateParts.join(' | '));
			}
			
			return parts.join('\n');
		}

		function formatEdgeDataTransfer(dt, pipelineType) {
			if (!dt) return '';
			let parts = [];
			const rateParts = [];
			
			if (pipelineType === 'traces' || !pipelineType) {
				const inputRate = dt.spans_input_rate || 0;
				const outputRate = dt.spans_transferred_rate || 0;
				if (inputRate > 0 || outputRate > 0) {
					if (inputRate > 0 && outputRate > 0 && Math.abs(inputRate - outputRate) > 0.01) {
						rateParts.push(formatRate(inputRate) + ' --> ' + formatRate(outputRate) + ' spans');
					} else if (outputRate > 0) {
						rateParts.push(formatRate(outputRate) + ' spans');
					} else if (inputRate > 0) {
						rateParts.push(formatRate(inputRate) + ' spans');
					}
				}
			}
			if (pipelineType === 'logs' || !pipelineType) {
				const inputRate = dt.logs_input_rate || 0;
				const outputRate = dt.logs_transferred_rate || 0;
				if (inputRate > 0 || outputRate > 0) {
					if (inputRate > 0 && outputRate > 0 && Math.abs(inputRate - outputRate) > 0.01) {
						rateParts.push(formatRate(inputRate) + ' --> ' + formatRate(outputRate) + ' logs');
					} else if (outputRate > 0) {
						rateParts.push(formatRate(outputRate) + ' logs');
					} else if (inputRate > 0) {
						rateParts.push(formatRate(inputRate) + ' logs');
					}
				}
			}
			if (pipelineType === 'metrics' || !pipelineType) {
				const inputRate = dt.metrics_input_rate || 0;
				const outputRate = dt.metrics_transferred_rate || 0;
				if (inputRate > 0 || outputRate > 0) {
					if (inputRate > 0 && outputRate > 0 && Math.abs(inputRate - outputRate) > 0.01) {
						rateParts.push(formatRate(inputRate) + ' --> ' + formatRate(outputRate) + ' metrics');
					} else if (outputRate > 0) {
						rateParts.push(formatRate(outputRate) + ' metrics');
					} else if (inputRate > 0) {
						rateParts.push(formatRate(inputRate) + ' metrics');
					}
				}
			}
			if (rateParts.length > 0) {
				parts.push('⚡ ' + rateParts.join(', '));
			}
			
			return parts.join(' | ');
		}

		function getTotalRate(dt, pipelineType) {
			if (!dt) return 0;
			let totalRate = 0;
			if (pipelineType === 'traces' || !pipelineType) {
				totalRate += dt.spans_transferred_rate || 0;
			}
			if (pipelineType === 'logs' || !pipelineType) {
				totalRate += dt.logs_transferred_rate || 0;
			}
			if (pipelineType === 'metrics' || !pipelineType) {
				totalRate += dt.metrics_transferred_rate || 0;
			}
			return totalRate;
		}

		function renderGraph() {
			const graph = document.getElementById('graph');
			if (!graph) return;
			
			if (graph.offsetWidth === 0 || graph.offsetHeight === 0) {
				graph.style.width = '100%';
				graph.style.height = '600px';
			}
			
			graph.innerHTML = '<svg id="edges"><defs><marker id="arrowhead" markerWidth="8" markerHeight="8" refX="7" refY="2.5" orient="auto"><polygon points="0 0, 8 2.5, 0 5" fill="#9ca3af" /></marker></defs></svg>';

			if (nodes.length === 0) {
				document.getElementById('info').textContent = 'No nodes found. Make sure agents are connected.';
				return;
			}

			loadSavedPositions();
			calculateLayout();
			renderEdges();

			let renderedCount = 0;
			nodes.forEach((node, index) => {
				let pos = nodePositions[node.id];
				if (!pos) {
					const defaultX = 80 + (index % 5) * 130;
					const defaultY = 80 + Math.floor(index / 5) * 90;
					pos = { x: defaultX, y: defaultY };
					nodePositions[node.id] = pos;
				}

				const nodeDiv = document.createElement('div');
				nodeDiv.className = 'node ' + node.type;
				nodeDiv.id = node.id;
				// Adjust positioning for compact boxes
				nodeDiv.style.left = (pos.x - 50) + 'px';
				nodeDiv.style.top = (pos.y - 15) + 'px';
				
				// Create main name element
				const nameSpan = document.createElement('span');
				nameSpan.style.display = 'block';
				nameSpan.style.fontSize = '13px';
				nameSpan.style.fontWeight = '600';
				nameSpan.style.marginBottom = '2px';
				nameSpan.textContent = node.name || node.id;
				nodeDiv.appendChild(nameSpan);
				
				if (node.pipeline_type) {
					const label = document.createElement('span');
					label.className = 'node-label';
					label.textContent = node.pipeline_type;
					nodeDiv.appendChild(label);
				}
				
				if (node.data_transfer) {
					const dtStr = formatNodeDataTransfer(node.data_transfer, node.pipeline_type);
					if (dtStr) {
						const rateLabel = document.createElement('div');
						rateLabel.className = 'rate-label';
						rateLabel.style.color = '#FFFFFF';
						rateLabel.style.fontSize = '10px';
						rateLabel.style.lineHeight = '1.3';
						rateLabel.innerHTML = dtStr.replace('⚡ ', '').replace(/\|/g, '<br>');
						nodeDiv.appendChild(rateLabel);
					}
				}
				
				makeNodeDraggable(nodeDiv, node.id);
				graph.appendChild(nodeDiv);
				renderedCount++;
			});
		}

		function makeNodeDraggable(nodeElement, nodeId) {
			nodeElement.addEventListener('mousedown', function(e) {
				if (e.button !== 0) return;
				
				isDragging = true;
				draggedNode = nodeElement;
				draggedNode.classList.add('dragging');
				
				const rect = nodeElement.getBoundingClientRect();
				const graphRect = document.getElementById('graph').getBoundingClientRect();
				
				const nodeCenterX = rect.left - graphRect.left + rect.width / 2;
				const nodeCenterY = rect.top - graphRect.top + rect.height / 2;
				
				dragOffset.x = e.clientX - graphRect.left - nodeCenterX;
				dragOffset.y = e.clientY - graphRect.top - nodeCenterY;
				
				e.preventDefault();
				e.stopPropagation();
			});
		}

		document.addEventListener('mousemove', function(e) {
			if (!isDragging || !draggedNode) return;
			
			const graph = document.getElementById('graph');
			const graphRect = graph.getBoundingClientRect();
			
			const newX = e.clientX - graphRect.left - dragOffset.x;
			const newY = e.clientY - graphRect.top - dragOffset.y;
			
			draggedNode.style.left = (newX - 50) + 'px';
			draggedNode.style.top = (newY - 15) + 'px';
			
			const nodeId = draggedNode.id;
			nodePositions[nodeId] = {
				x: newX,
				y: newY
			};
			
			renderEdges();
		});

		document.addEventListener('mouseup', function(e) {
			if (isDragging && draggedNode) {
				isDragging = false;
				draggedNode.classList.remove('dragging');
				draggedNode = null;
				savePositions();
			}
		});

		function renderEdges() {
			const svg = document.getElementById('edges');
			if (!svg) return;
			
			const existingLines = svg.querySelectorAll('line');
			const existingTexts = svg.querySelectorAll('text');
			const existingRects = svg.querySelectorAll('rect');
			existingLines.forEach(el => el.remove());
			existingTexts.forEach(el => el.remove());
			existingRects.forEach(el => el.remove());
			
			const graph = document.getElementById('graph');
			svg.setAttribute('width', graph.offsetWidth);
			svg.setAttribute('height', graph.offsetHeight);
			
			edges.forEach(edge => {
				const fromNode = nodePositions[edge.from];
				const toNode = nodePositions[edge.to];
				if (fromNode && toNode) {
					const nodeHeight = 35;
					const nodeWidth = 100;
					
					const dx = toNode.x - fromNode.x;
					const dy = toNode.y - fromNode.y;
					const distance = Math.sqrt(dx * dx + dy * dy);
					
					if (distance === 0) return;
					
					const unitX = dx / distance;
					const unitY = dy / distance;
					
					const startX = fromNode.x + (unitX * nodeWidth / 2);
					const startY = fromNode.y + nodeHeight / 2 + (unitY * nodeHeight / 2);
					const endX = toNode.x - (unitX * nodeWidth / 2);
					const endY = toNode.y - nodeHeight / 2 - (unitY * nodeHeight / 2);
					
					const line = document.createElementNS('http://www.w3.org/2000/svg', 'line');
					line.setAttribute('x1', startX);
					line.setAttribute('y1', startY);
					line.setAttribute('x2', endX);
					line.setAttribute('y2', endY);
					
					const colors = {
						'logs': '#f59e0b',
						'metrics': '#10b981',
						'traces': '#3b82f6'
					};
					line.setAttribute('stroke', colors[edge.pipeline_type] || '#9ca3af');
					
					let strokeWidth = 1.5;
					const totalRate = getTotalRate(edge.data_transfer, edge.pipeline_type);
					if (totalRate > 0) {
						const logValue = Math.log10(Math.max(1, totalRate));
						strokeWidth = Math.min(3, Math.max(1.5, 1.5 + logValue * 0.3));
					}
					line.setAttribute('stroke-width', strokeWidth);
					line.setAttribute('marker-end', 'url(#arrowhead)');
					
					const dt = edge.data_transfer;
					let hasInputOrOutputRate = false;
					if (dt) {
						const pipelineType = edge.pipeline_type;
						if (pipelineType === 'traces' || !pipelineType) {
							hasInputOrOutputRate = hasInputOrOutputRate || (dt.spans_input_rate > 0 || dt.spans_transferred_rate > 0);
						}
						if (pipelineType === 'logs' || !pipelineType) {
							hasInputOrOutputRate = hasInputOrOutputRate || (dt.logs_input_rate > 0 || dt.logs_transferred_rate > 0);
						}
						if (pipelineType === 'metrics' || !pipelineType) {
							hasInputOrOutputRate = hasInputOrOutputRate || (dt.metrics_input_rate > 0 || dt.metrics_transferred_rate > 0);
						}
					}
					
					if (edge.data_transfer && (totalRate > 0 || hasInputOrOutputRate)) {
						const midX = (startX + endX) / 2;
						const midY = (startY + endY) / 2;
						let labelText = '';
						let showInputOutput = false;
						
						const pipelineType = edge.pipeline_type;
						if (pipelineType === 'traces' || !pipelineType) {
							const inputRate = dt.spans_input_rate || 0;
							const outputRate = dt.spans_transferred_rate || 0;
							if (inputRate > 0 && outputRate > 0 && Math.abs(inputRate - outputRate) > 0.01) {
								labelText = formatRate(inputRate) + ' --> ' + formatRate(outputRate) + ' spans';
								showInputOutput = true;
							} else if (outputRate > 0) {
								labelText = formatRate(outputRate) + ' spans';
							} else if (inputRate > 0) {
								labelText = formatRate(inputRate) + ' spans';
							}
						}
						if (pipelineType === 'logs' || !pipelineType) {
							const inputRate = dt.logs_input_rate || 0;
							const outputRate = dt.logs_transferred_rate || 0;
							if (inputRate > 0 && outputRate > 0 && Math.abs(inputRate - outputRate) > 0.01) {
								labelText = formatRate(inputRate) + ' --> ' + formatRate(outputRate) + ' logs';
								showInputOutput = true;
							} else if (outputRate > 0) {
								labelText = formatRate(outputRate) + ' logs';
							} else if (inputRate > 0) {
								labelText = formatRate(inputRate) + ' logs';
							}
						}
						if (pipelineType === 'metrics' || !pipelineType) {
							const inputRate = dt.metrics_input_rate || 0;
							const outputRate = dt.metrics_transferred_rate || 0;
							if (inputRate > 0 && outputRate > 0 && Math.abs(inputRate - outputRate) > 0.01) {
								labelText = formatRate(inputRate) + ' --> ' + formatRate(outputRate) + ' metrics';
								showInputOutput = true;
							} else if (outputRate > 0) {
								labelText = formatRate(outputRate) + ' metrics';
							} else if (inputRate > 0) {
								labelText = formatRate(inputRate) + ' metrics';
							}
						}
						
						if (!labelText) {
							labelText = formatEdgeDataTransfer(edge.data_transfer, edge.pipeline_type).replace('⚡ ', '');
						}
						
						if (labelText) {
							const text = document.createElementNS('http://www.w3.org/2000/svg', 'text');
							text.setAttribute('x', midX);
							text.setAttribute('y', midY - 5);
							text.setAttribute('text-anchor', 'middle');
							text.setAttribute('font-size', showInputOutput ? '8px' : '9px');
							text.setAttribute('fill', showInputOutput ? '#FF6B35' : '#4CAF50');
							text.setAttribute('pointer-events', 'none');
							text.style.fontWeight = 'bold';
							text.textContent = labelText;
							
							const bg = document.createElementNS('http://www.w3.org/2000/svg', 'rect');
							const charWidth = showInputOutput ? 4.5 : 5.5;
							const textWidth = labelText.length * charWidth;
							bg.setAttribute('x', midX - textWidth / 2 - 3);
							bg.setAttribute('y', midY - 13);
							bg.setAttribute('width', textWidth + 6);
							bg.setAttribute('height', showInputOutput ? '16' : '14');
							bg.setAttribute('fill', showInputOutput ? '#FFF8F0' : 'white');
							bg.setAttribute('opacity', '0.95');
							bg.setAttribute('rx', '3');
							bg.setAttribute('stroke', showInputOutput ? '#FF6B35' : 'none');
							bg.setAttribute('stroke-width', showInputOutput ? '1' : '0');
							
							svg.appendChild(bg);
							svg.appendChild(text);
						}
					}
					
					svg.appendChild(line);
				}
			});
		}

		if (document.readyState === 'loading') {
			document.addEventListener('DOMContentLoaded', loadTopology);
		} else {
			loadTopology();
		}
	</script>
</body>
</html>`
}
