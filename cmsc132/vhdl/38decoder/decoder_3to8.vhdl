LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY decoder_3to8 IS
   PORT (
      I2, I1, I0: IN STD_LOGIC);
      Out7, Out6, Out5, Out4, Out3, Out2, Out1, Out0: OUT STD_LOGIC;
   );
END decoder_3to8;
--------------------------------

ARCHITECTURE pure_logic OF decoder_3to8 IS
BEGIN
   Out0 <= NOT INPUT(2) AND NOT INPUT(1) and NOT INPUT(0);
   Out1 <= NOT INPUT(2) AND NOT INPUT(1) and INPUT(0);
   Out2 <= NOT INPUT(2) AND INPUT(1) and NOT INPUT(0);
   Out3 <= NOT INPUT(2) AND INPUT(1) and INPUT(0);
   Out4 <= INPUT(2) AND NOT INPUT(1) and NOT INPUT(0);
   Out5 <= INPUT(2) AND NOT INPUT(1) and INPUT(0);
   Out6 <= INPUT(2) AND INPUT(1) and NOT INPUT(0);
   Out7 <= INPUT(2) AND INPUT(1) and INPUT(0);
END pure_logic;
