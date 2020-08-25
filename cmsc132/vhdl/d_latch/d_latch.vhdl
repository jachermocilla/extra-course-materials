LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY d_latch IS
   PORT (C, D: IN STD_LOGIC;
   Q, Q_BAR: INOUT STD_LOGIC);
END d_latch;
--------------------------------
ARCHITECTURE pure_logic OF d_latch IS
BEGIN
   Q <= (C AND NOT D) NOR Q_BAR;
   Q_BAR <= (D AND C) NOR Q;
END pure_logic;
