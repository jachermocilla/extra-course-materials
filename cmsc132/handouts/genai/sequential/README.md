## Convert to PDF
pandoc seque.md -o seque.pdf --pdf-engine=xelatex -V mainfont="DejaVu Serif"
pandoc seque_vhdl.md -o seque_vhdl.pdf --pdf-engine=xelatex -V mainfont="DejaVu Serif"

