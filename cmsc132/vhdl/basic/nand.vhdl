LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY nand_gate IS
   PORT (A, B: IN STD_LOGIC;
   C: OUT STD_LOGIC);
END nand_gate;
--------------------------------

ARCHITECTURE pure_logic OF nand_gate IS
BEGIN
   C <= A  NAND B;
END pure_logic;
