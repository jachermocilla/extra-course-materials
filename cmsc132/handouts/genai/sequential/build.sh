#!/bin/bash
pandoc seque.md -o seque.pdf --pdf-engine=xelatex -V mainfont="DejaVu Serif"
pandoc seque_vhdl.md -o seque_vhdl.pdf --pdf-engine=xelatex -V mainfont="DejaVu Serif"
pandoc counter_design.md -o counter_design.pdf --pdf-engine=xelatex -V mainfont="DejaVu Serif" -V monofont="DejaVu Sans Mono" 

