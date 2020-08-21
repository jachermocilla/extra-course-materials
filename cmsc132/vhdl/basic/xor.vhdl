LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY xor_gate IS
   PORT (A, B: IN STD_LOGIC;
   C: OUT STD_LOGIC);
END xor_gate;
--------------------------------

ARCHITECTURE pure_logic OF xor_gate IS
BEGIN
   C <= A  XOR B;
END pure_logic;
