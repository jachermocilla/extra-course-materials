#!/bin/bash

# SCHED_FIFO Experiment Script
# This script demonstrates the difference between SCHED_FIFO and SCHED_OTHER policies

echo "================================================"
echo "Linux SCHED_FIFO Policy Experiment"
echo "================================================"

# Compile the test program
echo "Compiling test program..."
gcc -o sched_test sched_test.c -Wall

if [ $? -ne 0 ]; then
    echo "Compilation failed!"
    exit 1
fi

echo "Compilation successful!"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "WARNING: Not running as root. SCHED_FIFO may not work."
    echo "To see full effects, run: sudo $0"
    echo ""
fi

echo "================================================"
echo "Test 1: Running with SCHED_OTHER (default policy)"
echo "================================================"

echo "Starting 3 processes with SCHED_OTHER policy..."
./sched_test 0 2000 &
PID1=$!
./sched_test 0 2000 &
PID2=$!
./sched_test 0 2000 &
PID3=$!

# Wait for all processes to complete
wait $PID1 $PID2 $PID3

echo ""
echo "================================================"
echo "Test 2: Running with SCHED_FIFO (real-time policy)"
echo "================================================"

if [ "$EUID" -eq 0 ]; then
    echo "Starting 3 processes with SCHED_FIFO policy..."
    echo "Note: Higher priority processes should complete first"
    
    # Start processes with different priorities
    echo "Starting high priority process (priority 50)..."
    ./sched_test 50 2000 &
    HIGH_PID=$!
    
    sleep 0.1  # Small delay to ensure order
    
    echo "Starting medium priority process (priority 30)..."
    ./sched_test 30 2000 &
    MED_PID=$!
    
    sleep 0.1
    
    echo "Starting low priority process (priority 10)..."
    ./sched_test 10 2000 &
    LOW_PID=$!
    
    # Wait for all processes
    wait $HIGH_PID $MED_PID $LOW_PID
else
    echo "Skipping SCHED_FIFO test - requires root privileges"
    echo "Run 'sudo ./run_experiment.sh' to see SCHED_FIFO in action"
fi

echo ""
echo "================================================"
echo "Test 3: Mixed Policy Comparison"
echo "================================================"

if [ "$EUID" -eq 0 ]; then
    echo "Starting mixed processes: SCHED_FIFO vs SCHED_OTHER"
    
    # Start a regular process
    ./sched_test 0 3000 &
    REGULAR_PID=$!
    
    sleep 0.5  # Let regular process start
    
    # Start a high-priority real-time process
    echo "Injecting high-priority SCHED_FIFO process..."
    ./sched_test 80 3000 &
    RT_PID=$!
    
    wait $REGULAR_PID $RT_PID
else
    echo "Skipping mixed policy test - requires root privileges"
fi

echo ""
echo "================================================"
echo "Experiment Complete!"
echo "================================================"

echo "Key Observations:"
echo "1. SCHED_OTHER processes share CPU time fairly"
echo "2. SCHED_FIFO processes run to completion based on priority"
echo "3. Higher priority SCHED_FIFO processes preempt lower priority ones"
echo "4. SCHED_FIFO processes can preempt SCHED_OTHER processes"
echo ""
echo "To see scheduling in real-time, you can also run:"
echo "  htop or top in another terminal while running this experiment"

# Cleanup
rm -f sched_test

echo "Cleanup complete."
