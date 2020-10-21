#!/bin/bash

for f in *.vhd; do
  ghdl -a $f
  ghdl -e "${f%%.*}"
done
  
ghdl -a Processor_tb.vhdl
ghdl -e Processor_tb
ghdl -r  Processor_tb  --vcd=minicpu.vcd  --stop-time=500ns
