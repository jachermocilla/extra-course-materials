## VHDL and GTKWave

```shell
$ghdl -s dff.vhd dff_tb.vhd
$ghdl -a dff.vhd dff_tb.vhd
$ghdl -e dff
$ghdl -r dff_tb --vcd=dff.vcd --stop-time=500ns
$gtkwave dff.vcd
```

[How?](http://enos.itcollege.ee/~vpesonen/lisa/lab_example.html)

