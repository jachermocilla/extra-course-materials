LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY sr_latch IS
   PORT (R, S: IN STD_LOGIC;
   Q, Q_BAR: INOUT STD_LOGIC);
END sr_latch;
--------------------------------
ARCHITECTURE pure_logic OF sr_latch IS
BEGIN
   Q <= R NOR Q_BAR;
   Q_BAR <= S NOR Q;
END pure_logic;
