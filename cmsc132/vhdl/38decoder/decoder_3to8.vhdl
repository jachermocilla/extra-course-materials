LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY decoder_3to8 IS
   PORT (
      INPUT: IN STD_LOGIC_VECTOR(2 downto 0);
      OUTPUT: OUT STD_LOGIC_VECTOR(7 downto 0)
   );
END decoder_3to8;
--------------------------------

ARCHITECTURE pure_logic OF decoder_3to8 IS
BEGIN
   OUTPUT(0) <= NOT INPUT(2) AND NOT INPUT(1) and NOT INPUT(0);
   OUTPUT(1) <= NOT INPUT(2) AND NOT INPUT(1) and INPUT(0);
   OUTPUT(2) <= NOT INPUT(2) AND INPUT(1) and NOT INPUT(0);
   OUTPUT(3) <= NOT INPUT(2) AND INPUT(1) and INPUT(0);
   OUTPUT(4) <= INPUT(2) AND NOT INPUT(1) and NOT INPUT(0);
   OUTPUT(5) <= INPUT(2) AND NOT INPUT(1) and INPUT(0);
   OUTPUT(6) <= INPUT(2) AND INPUT(1) and NOT INPUT(0);
   OUTPUT(7) <= INPUT(2) AND INPUT(1) and INPUT(0);
END pure_logic;
