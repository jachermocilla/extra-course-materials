LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY and_gate IS
   PORT (A, B: IN STD_LOGIC;
   C: OUT STD_LOGIC);
END and_gate;
--------------------------------

ARCHITECTURE pure_logic OF and_gate IS
BEGIN
   C <= A  AND B;
END pure_logic;
