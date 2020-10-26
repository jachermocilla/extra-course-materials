# REDHORSE 500 
(Original: https://github.com/jirawatsaesu/8-Bit-Simple-Processor)

FPGA-based 8-bit simple processor designed using the VHDL language

Main components
* Program counter (PC)
* Can accommodate 8 Instructions with 8-bit instruction "word"
* 4 8-bit registers
* Arithmetic logic unit (ALU)
* Control unit

> [RTL view](https://drive.google.com/file/d/1rpwMigKBZJ3UliGxDvjsoxlWWDj-KijM/view)


## Program Counter (PC)
Send the address of current instruction to **Instruction Memory** and increase every falling edge of clock.
```
    if falling_edge(clock) then

        next_address <= current_address + 1

    end if
```

> [Program Counter RTL view](https://drive.google.com/file/d/1XNqxDJd4e8GmHNdur8K8svxxR0AMyshN/view)


## Instruction Memory
A memory unit to store the 8-bit instructions of a program. Fetch each instructions with address.
```
    instruction <= instruction_set(current_address)
```

R-type instruction splitted.
* 2-bit operation code send to **ALU** and **Control Unit**.
* 2-bit address of source register 1 send to **Registers File**.
* 2-bit address of source register 2 send to **Registers File**.
* 2-bit address of destination register send to **Registers File**.
```
    op_code <= instruction(7 downto 6)
    
    src1_address <= instruction(5 downto 4)
    
    src2_address <= instruction(3 downto 2)
    
    dst_address <= instruction(1 downto 0)
```

> [Instruction Memory RTL view](https://drive.google.com/file/d/1AhVxxtvg4UsZmPhP0bRNyDSOaRBW7gFf/view)


## Registers File
4 8-bit registers to store the binary from ``00000000`` to ``11111111`` (default first register is ``00000001``, second register is ``00000010`` and so on). Fetch each registers with address.

* 8-bit value of source register 1 and 2 send to **ALU**.
```
    src1_value <= registers(src1_address)
    
    src2_value <= registers(src2_address)
```

* Write 8-bit data result from **ALU** to destination register every falling edge of clock.
```
    if falling_edge(clock) then

        registers(dst_address) <= result

    end if
```

> [Registers File RTL view](https://drive.google.com/file/d/1AiNfngFQ1bxOaHfD-mCPGkiRUFbKa6zX/view)


## Arithmetic Logic Unit (ALU)
Calculates arithmetic operations then sent to **Registers File**. Select each operation with operation code.

* operation code ``00`` is ``and``.
* operation code ``01`` is ``add`` (addition).
* operation code ``10`` is ``sub`` (subtraction).
* operation code ``11`` is ``addi`` (addition immediate).
```
    if (op_code = "00") then
      result <= src1_value and src2_value

    elsif (op_code = "01") then
      result <= src1_value + src2_value

    elsif (op_code = "10") then
      result <= src1_value - src2_value

    elsif (op_code = "11") then
      result <= src1_value + data

    end if
```

> [ALU  RTL view](https://drive.google.com/file/d/18Chcsz0xrmOXY5ilLNuj2NSX1TgjxEPJ/view)


## Sign Extend
Special component for ``addi`` operation. Extend bits of data from 2 to 8 then send to **ALU**.
```
    # positive value only
    data_out <= "000000" & data_in
```

> [Sign Extend RTL view](https://drive.google.com/file/d/19vgNB8Plq-mHS5CR3cvFRdd0sxRYl8yK/view)


## Mux
### Mux0
Special component for ``addi`` operation. Select destination address with ``reg_dst`` from **Control Unit** then send destination to **Registers File**.

* ``reg_dst`` is ``0``. Default destiantion address.
* ``reg_dst`` is ``1``. Change destination address, use address of source register 2.

### Mux1
Special component for ``addi`` operation. Select data with ``alu_src`` from **Control Unit** then sent data to **ALU**.

* ``alu_src`` is ``0``. Default data values from register
* ``alu_src`` is ``1``. Change data values, use data from **Sign Extend**


## Control Unit
Special component for ``addi`` operation. Control others component with operation code.

> ``reg_dst`` is *register destination control*.

> ``alu_src`` is *ALU source register data control*.

* ``op_code`` is ``11``.
    * ``reg_dst`` is ``1``.
    * ``alu_src`` is ``1``.
* Others.
    * ``reg_dst`` is ``0``.
    * ``alu_src`` is ``0``.

> [Control Unit RTL view](https://drive.google.com/file/d/1tc4FtgEREi3qc2BDbBXz5KbFWqkzp_Oj/view)
