LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY decoder_1to2 IS
   PORT (I: IN STD_LOGIC;
   Out0, Out1: OUT STD_LOGIC);
END decoder_1to2;
--------------------------------

ARCHITECTURE pure_logic OF decoder_1to2 IS
BEGIN
   PROCESS(I)
   BEGIN
      IF (I='0') THEN
         Out0 <= '1'; Out1 <= '0';
      ELSIF (I='0') THEN
         Out0 <= '0'; Out1 <= '1';
      END IF;
   END PROCESS;
END pure_logic;
