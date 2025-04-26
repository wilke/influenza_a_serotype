#!/bin/bash

# Benchmark and comparison script for influenza_a_serotype
# This script runs both the R and Go implementations and compares their performance and results

# Set variables
PAF_FILE="test_data/small_test.paf"
MAPPING_FILE="DBs/v1.25/Influenza_A_segment_info1.tsv"
SAMPLE_NAME="benchmark_test"
OUTPUT_DIR="test_output"
R_SCRIPT="src/iav_serotype/parse_pafs_influenza_A.R"
R_OUTPUT_DIR="${OUTPUT_DIR}/r_output"
SCORE_THRESHOLD="0.8"
ITERATIONS=3

# Create output directory
mkdir -p "$OUTPUT_DIR"
mkdir -p "$R_OUTPUT_DIR"

echo "===== Influenza A Serotype Benchmark ====="
echo "PAF File: $PAF_FILE"
echo "Mapping File: $MAPPING_FILE"
echo "Sample Name: $SAMPLE_NAME"
echo "Output Directory: $OUTPUT_DIR"
echo "R Script: $R_SCRIPT"
echo "R Output Directory: $R_OUTPUT_DIR"
echo "Score Threshold: $SCORE_THRESHOLD"
echo "Iterations: $ITERATIONS"
echo "========================================"

# Run benchmark
echo -e "\n===== Running Benchmark ====="
./src/paf2serotypes/paf2serotypes benchmark \
  --paf "$PAF_FILE" \
  --mapping "$MAPPING_FILE" \
  --sample "$SAMPLE_NAME" \
  --output "$OUTPUT_DIR" \
  --min-score "$SCORE_THRESHOLD" \
  --r-script "$R_SCRIPT" \
  --r-output "$R_OUTPUT_DIR" \
  --iterations "$ITERATIONS" \
  --report-file "${OUTPUT_DIR}/benchmark_report.json"

# Run comparison
echo -e "\n===== Running Comparison ====="
./src/paf2serotypes/paf2serotypes compare \
  --paf "$PAF_FILE" \
  --mapping "$MAPPING_FILE" \
  --sample "$SAMPLE_NAME" \
  --output "$OUTPUT_DIR" \
  --min-score "$SCORE_THRESHOLD" \
  --r-script "$R_SCRIPT" \
  --r-output "$R_OUTPUT_DIR" \
  --report-file "${OUTPUT_DIR}/comparison_report.json"

# Display summary
echo -e "\n===== Benchmark Summary ====="
echo "Benchmark report saved to: ${OUTPUT_DIR}/benchmark_report.json"
echo "Comparison report saved to: ${OUTPUT_DIR}/comparison_report.json"
echo "Go output saved to: ${OUTPUT_DIR}/${SAMPLE_NAME}_read_summary.tsv"
echo "R output saved to: ${R_OUTPUT_DIR}/${SAMPLE_NAME}_read_summary.tsv"
echo "========================================"

# Generate a simple HTML report
cat > "${OUTPUT_DIR}/benchmark_report.html" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>Influenza A Serotype Benchmark Report</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        h1, h2 { color: #333; }
        .container { max-width: 1000px; margin: 0 auto; }
        .card { border: 1px solid #ddd; border-radius: 5px; padding: 20px; margin-bottom: 20px; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 8px; text-align: left; border-bottom: 1px solid #ddd; }
        th { background-color: #f2f2f2; }
        .highlight { background-color: #e6f7ff; }
        .chart { height: 400px; margin: 20px 0; }
    </style>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
</head>
<body>
    <div class="container">
        <h1>Influenza A Serotype Benchmark Report</h1>
        
        <div class="card">
            <h2>Configuration</h2>
            <table>
                <tr><th>Parameter</th><th>Value</th></tr>
                <tr><td>PAF File</td><td>${PAF_FILE}</td></tr>
                <tr><td>Mapping File</td><td>${MAPPING_FILE}</td></tr>
                <tr><td>Sample Name</td><td>${SAMPLE_NAME}</td></tr>
                <tr><td>Score Threshold</td><td>${SCORE_THRESHOLD}</td></tr>
                <tr><td>Iterations</td><td>${ITERATIONS}</td></tr>
            </table>
        </div>
        
        <div class="card">
            <h2>Performance Comparison</h2>
            <div class="chart">
                <canvas id="performanceChart"></canvas>
            </div>
        </div>
        
        <div class="card">
            <h2>Memory Usage</h2>
            <div class="chart">
                <canvas id="memoryChart"></canvas>
            </div>
        </div>
        
        <div class="card">
            <h2>Output Comparison</h2>
            <div class="chart">
                <canvas id="outputChart"></canvas>
            </div>
        </div>
    </div>
    
    <script>
        // Load the JSON data
        fetch('benchmark_report.json')
            .then(response => response.json())
            .then(benchData => {
                fetch('comparison_report.json')
                    .then(response => response.json())
                    .then(compData => {
                        renderCharts(benchData, compData);
                    });
            });
            
        function renderCharts(benchData, compData) {
            // Performance chart
            const perfCtx = document.getElementById('performanceChart').getContext('2d');
            new Chart(perfCtx, {
                type: 'bar',
                data: {
                    labels: ['Runtime (ms)', 'CPU Usage (%)', 'I/O Operations'],
                    datasets: [
                        {
                            label: 'Go Implementation',
                            data: [
                                parseFloat(benchData.summary.avg_go_runtime.replace(/[^\d.-]/g, '')),
                                benchData.summary.avg_go_cpu,
                                benchData.go_results[0].io_stats.read_count + benchData.go_results[0].io_stats.write_count
                            ],
                            backgroundColor: 'rgba(54, 162, 235, 0.5)',
                            borderColor: 'rgba(54, 162, 235, 1)',
                            borderWidth: 1
                        },
                        {
                            label: 'R Implementation',
                            data: [
                                parseFloat(benchData.summary.avg_r_runtime.replace(/[^\d.-]/g, '')),
                                benchData.summary.avg_r_cpu,
                                benchData.r_results[0].io_stats.read_count + benchData.r_results[0].io_stats.write_count
                            ],
                            backgroundColor: 'rgba(255, 99, 132, 0.5)',
                            borderColor: 'rgba(255, 99, 132, 1)',
                            borderWidth: 1
                        }
                    ]
                },
                options: {
                    responsive: true,
                    scales: {
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'Value'
                            }
                        }
                    },
                    plugins: {
                        title: {
                            display: true,
                            text: 'Performance Metrics Comparison'
                        }
                    }
                }
            });
            
            // Memory chart
            const memCtx = document.getElementById('memoryChart').getContext('2d');
            new Chart(memCtx, {
                type: 'bar',
                data: {
                    labels: ['RSS (MB)', 'VMS (MB)'],
                    datasets: [
                        {
                            label: 'Go Implementation',
                            data: [
                                benchData.go_results[0].memory_usage.rss / (1024 * 1024),
                                benchData.go_results[0].memory_usage.vms / (1024 * 1024)
                            ],
                            backgroundColor: 'rgba(54, 162, 235, 0.5)',
                            borderColor: 'rgba(54, 162, 235, 1)',
                            borderWidth: 1
                        },
                        {
                            label: 'R Implementation',
                            data: [
                                benchData.r_results[0].memory_usage.rss / (1024 * 1024),
                                benchData.r_results[0].memory_usage.vms / (1024 * 1024)
                            ],
                            backgroundColor: 'rgba(255, 99, 132, 0.5)',
                            borderColor: 'rgba(255, 99, 132, 1)',
                            borderWidth: 1
                        }
                    ]
                },
                options: {
                    responsive: true,
                    scales: {
                        y: {
                            beginAtZero: true,
                            title: {
                                display: true,
                                text: 'Memory (MB)'
                            }
                        }
                    },
                    plugins: {
                        title: {
                            display: true,
                            text: 'Memory Usage Comparison'
                        }
                    }
                }
            });
            
            // Output comparison chart
            const outputCtx = document.getElementById('outputChart').getContext('2d');
            new Chart(outputCtx, {
                type: 'pie',
                data: {
                    labels: ['Matching Lines', 'Different Lines'],
                    datasets: [{
                        data: [compData.matching_lines, compData.different_lines],
                        backgroundColor: [
                            'rgba(75, 192, 192, 0.5)',
                            'rgba(255, 99, 132, 0.5)'
                        ],
                        borderColor: [
                            'rgba(75, 192, 192, 1)',
                            'rgba(255, 99, 132, 1)'
                        ],
                        borderWidth: 1
                    }]
                },
                options: {
                    responsive: true,
                    plugins: {
                        title: {
                            display: true,
                            text: 'Output Comparison'
                        },
                        legend: {
                            position: 'bottom'
                        }
                    }
                }
            });
        }
    </script>
</body>
</html>
EOF

echo "HTML report generated: ${OUTPUT_DIR}/benchmark_report.html"
echo "Open the HTML report to view detailed benchmark results"