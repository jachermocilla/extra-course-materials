pandoc --pdf-engine=xelatex -t beamer -V aspectratio=169 slides.md -o slides.pdf -V mainfont="Liberation Serif" -V monofont="DejaVu Sans Mono" --highlight-style zenburn 
