#!/bin/bash
#pandoc ${1}.md -o ${1}.pdf --pdf-engine=xelatex --template "./eisvogel.tex" -V mainfont="DejaVu Serif" -V monofont="DejaVu Sans Mono" 
pandoc ${1}.md -o ${1}.pdf --pdf-engine=xelatex --template "./eisvogel.tex" -V mainfont="Liberation Serif" -V monofont="Liberation Mono" && \
evince ${1}.pdf

