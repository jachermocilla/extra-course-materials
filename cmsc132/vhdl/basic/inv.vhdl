LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY inv_gate IS
   PORT (A: IN STD_LOGIC;
   B: OUT STD_LOGIC);
END inv_gate;
--------------------------------

ARCHITECTURE pure_logic OF inv_gate IS
BEGIN
   B <= NOT A;
END pure_logic;
