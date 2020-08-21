LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY nor_gate IS
   PORT (A, B: IN STD_LOGIC;
   C: OUT STD_LOGIC);
END nor_gate;
--------------------------------

ARCHITECTURE pure_logic OF nor_gate IS
BEGIN
   C <= A  NOR B;
END pure_logic;
