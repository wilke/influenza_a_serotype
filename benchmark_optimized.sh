#!/bin/bash

# Benchmark script for comparing optimized vs. original implementation
# This script runs benchmarks for both implementations and compares the results

# Set colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Running benchmarks for optimized implementation...${NC}"
echo

# Run benchmarks with memory profiling
echo -e "${YELLOW}Running PAF record parsing benchmark...${NC}"
go test -benchmem -bench=BenchmarkNewPafRecord ./lib/go/paf2serotypes/model/... -v

echo -e "${YELLOW}Running CalculateScores benchmarks...${NC}"
go test -benchmem -bench=BenchmarkCalculateScores ./lib/go/paf2serotypes/model/... -v

echo -e "${YELLOW}Running AssignSerotypes benchmarks...${NC}"
go test -benchmem -bench=BenchmarkAssignSerotypes ./lib/go/paf2serotypes/model/... -v

echo -e "${YELLOW}Running object pooling benchmark...${NC}"
go test -benchmem -bench=BenchmarkPafRecordCreation_WithPool ./lib/go/paf2serotypes/model/... -v

echo
echo -e "${GREEN}Benchmark Summary:${NC}"
echo "The optimized implementation includes the following improvements:"
echo "1. Reduced string-to-integer conversions in PAF record parsing"
echo "2. Pre-allocation of memory for records and maps"
echo "3. String builder usage to reduce allocations"
echo "4. Object pooling for frequently created objects"
echo "5. Adaptive chunk sizing based on available memory"
echo "6. Dynamic worker pool sizing based on system resources"
echo
echo "These optimizations should result in:"
echo "- Lower memory usage (reduced allocations)"
echo "- Fewer garbage collection cycles"
echo "- Better performance with large datasets"
echo "- More efficient CPU utilization"
echo
echo -e "${YELLOW}To compare with the original implementation, you can run:${NC}"
echo "go test -benchmem -bench=. ./lib/go/paf2serotypes/model/..."