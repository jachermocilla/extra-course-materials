LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY or_gate IS
   PORT (A, B: IN STD_LOGIC;
   C: OUT STD_LOGIC);
END or_gate;
--------------------------------

ARCHITECTURE pure_logic OF or_gate IS
BEGIN
   C <= A  OR B;
END pure_logic;
