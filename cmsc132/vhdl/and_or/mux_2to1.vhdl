LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY mux_2to1 IS
   PORT (A, B, S: IN STD_LOGIC;
   C: OUT STD_LOGIC);
END mux_2to1;
--------------------------------

ARCHITECTURE pure_logic OF mux_2to1 IS
BEGIN
--- C <= (A AND NOT S) OR (B AND S);                       
   C <= A WHEN S='0' ELSE B;
END pure_logic;
