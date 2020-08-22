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
   Out0 <= NOT I2 AND NOT I1 and NOT I0;
   Out1 <= NOT I2 AND NOT I1 and I0;
   Out2 <= NOT I2 AND I1 and NOT I0;
   Out3 <= NOT I2 AND I1 and I0;
   Out4 <= I2 AND NOT I1 and NOT I0;
   Out5 <= I2 AND NOT I1 and I0;
   Out6 <= I2 AND I1 and NOT I0;
   Out7 <= I2 AND I1 and I0;
END pure_logic;
