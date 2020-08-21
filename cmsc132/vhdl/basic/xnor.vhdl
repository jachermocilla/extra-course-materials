LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY xnor_gate IS
   PORT (A, B: IN STD_LOGIC;
   C: OUT STD_LOGIC);
END xnor_gate;
--------------------------------

ARCHITECTURE pure_logic OF xnor_gate IS
BEGIN
   C <= A  XNOR B;
END pure_logic;
