// Main JavaScript for MCP Coverage Visualization

// Initialize on DOM ready
document.addEventListener('DOMContentLoaded', function() {
    initializeCoverageColors();
    initializeInteractiveElements();
    setupAPIHandlers();
});

// Color coverage bars based on percentage
function initializeCoverageColors() {
    const coverageBars = document.querySelectorAll('.coverage-fill');
    coverageBars.forEach(bar => {
        const width = parseFloat(bar.style.width);
        if (width >= 80) {
            bar.style.backgroundColor = 'var(--coverage-high)';
        } else if (width >= 60) {
            bar.style.backgroundColor = 'var(--coverage-medium)';
        } else {
            bar.style.backgroundColor = 'var(--coverage-low)';
        }
    });
}

// Setup interactive elements
function initializeInteractiveElements() {
    // File browser
    const fileLinks = document.querySelectorAll('.file-link');
    fileLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            e.preventDefault();
            const path = this.dataset.path;
            loadFileView(path);
        });
    });
    
    // Test details
    const testLinks = document.querySelectorAll('.test-link');
    testLinks.forEach(link => {
        link.addEventListener('click', function(e) {
            if (!e.shiftKey && !e.metaKey) {
                e.preventDefault();
                const testId = this.dataset.testId || this.textContent;
                loadTestView(testId);
            }
        });
    });
}

// API handlers
function setupAPIHandlers() {
    window.mcpAPI = {
        getCoverage: async function() {
            const response = await fetch('/api/coverage');
            return response.json();
        },
        
        getFile: async function(path) {
            const response = await fetch(`/api/files/${encodeURIComponent(path)}`);
            return response.json();
        },
        
        getTests: async function() {
            const response = await fetch('/api/tests');
            return response.json();
        },
        
        getSessions: async function() {
            const response = await fetch('/api/sessions');
            return response.json();
        }
    };
}

// Load file view dynamically
async function loadFileView(path) {
    try {
        const file = await window.mcpAPI.getFile(path);
        renderFileView(file);
    } catch (error) {
        console.error('Failed to load file:', error);
    }
}

// Load test view dynamically
async function loadTestView(testId) {
    try {
        const tests = await window.mcpAPI.getTests();
        const test = tests.find(t => t.testName === testId);
        if (test) {
            renderTestView(test);
        }
    } catch (error) {
        console.error('Failed to load test:', error);
    }
}

// Render file view in modal or inline
function renderFileView(file) {
    // Implementation would depend on UI framework
    console.log('Render file view:', file);
}

// Render test view in modal or inline
function renderTestView(test) {
    // Implementation would depend on UI framework
    console.log('Render test view:', test);
}

// Timeline visualization enhancements
if (document.querySelector('.timeline-container')) {
    // Add zoom and pan functionality
    const timelines = document.querySelectorAll('.timeline');
    timelines.forEach(timeline => {
        let scale = 1;
        let translateX = 0;
        
        timeline.addEventListener('wheel', function(e) {
            e.preventDefault();
            const delta = e.deltaY > 0 ? 0.9 : 1.1;
            scale *= delta;
            scale = Math.max(0.5, Math.min(3, scale));
            this.style.transform = `scaleX(${scale}) translateX(${translateX}px)`;
        });
        
        let dragging = false;
        let startX = 0;
        
        timeline.addEventListener('mousedown', function(e) {
            dragging = true;
            startX = e.clientX - translateX;
            this.style.cursor = 'grabbing';
        });
        
        timeline.addEventListener('mousemove', function(e) {
            if (!dragging) return;
            translateX = e.clientX - startX;
            this.style.transform = `scaleX(${scale}) translateX(${translateX}px)`;
        });
        
        timeline.addEventListener('mouseup', function() {
            dragging = false;
            this.style.cursor = 'grab';
        });
    });
    
    // Add tooltips for trace markers
    const traceMarkers = document.querySelectorAll('.trace-marker');
    traceMarkers.forEach(marker => {
        marker.addEventListener('mouseenter', function(e) {
            const tooltip = document.createElement('div');
            tooltip.className = 'tooltip';
            tooltip.textContent = this.title;
            tooltip.style.position = 'absolute';
            tooltip.style.left = e.pageX + 'px';
            tooltip.style.top = (e.pageY - 30) + 'px';
            document.body.appendChild(tooltip);
            
            this.addEventListener('mouseleave', function() {
                tooltip.remove();
            });
        });
    });
}

// Export functionality
window.exportCoverageData = async function(format = 'json') {
    const coverage = await window.mcpAPI.getCoverage();
    
    switch (format) {
        case 'json':
            downloadFile('coverage.json', JSON.stringify(coverage, null, 2), 'application/json');
            break;
        case 'html':
            // Generate static HTML report
            const html = generateStaticHTMLReport(coverage);
            downloadFile('coverage.html', html, 'text/html');
            break;
        case 'csv':
            const csv = generateCSVReport(coverage);
            downloadFile('coverage.csv', csv, 'text/csv');
            break;
    }
};

// Utility functions
function downloadFile(filename, content, mimeType) {
    const blob = new Blob([content], { type: mimeType });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
    URL.revokeObjectURL(url);
}

function generateStaticHTMLReport(coverage) {
    // Simple HTML report generation
    return `
<!DOCTYPE html>
<html>
<head>
    <title>Coverage Report</title>
    <style>
        body { font-family: sans-serif; margin: 2rem; }
        .summary { background: #f0f0f0; padding: 1rem; margin-bottom: 2rem; }
        .file { margin-bottom: 1rem; }
        .covered { background: #90ee90; }
        .uncovered { background: #ffcccb; }
    </style>
</head>
<body>
    <h1>Coverage Report</h1>
    <div class="summary">
        <h2>Summary</h2>
        <p>Line Coverage: ${coverage.summary.coverage.line.toFixed(1)}%</p>
        <p>Files: ${coverage.summary.coveredFiles}/${coverage.summary.totalFiles}</p>
    </div>
    ${Object.entries(coverage.files).map(([path, file]) => `
        <div class="file">
            <h3>${path}</h3>
            <p>Coverage: ${file.coverage.coveragePercent.toFixed(1)}%</p>
        </div>
    `).join('')}
</body>
</html>
    `;
}

function generateCSVReport(coverage) {
    const rows = [
        ['File', 'Coverage %', 'Lines Covered', 'Total Lines']
    ];
    
    Object.entries(coverage.files).forEach(([path, file]) => {
        rows.push([
            path,
            file.coverage.coveragePercent.toFixed(1),
            file.coverage.coveredLines,
            file.coverage.totalLines
        ]);
    });
    
    return rows.map(row => row.join(',')).join('\n');
}