LIBRARY ieee;
USE ieee.std_logic_1164.ALL;

ENTITY dff is
   PORT( D: IN STD_LOGIC;
      C: IN STD_LOGIC;
      Q: INOUT STD_LOGIC;
      Q_BAR: INOUT STD_LOGIC);
END dff;

ARCHITECTURE behavioral OF dff IS
   COMPONENT d_latch IS
      PORT (C, D: IN STD_LOGIC;
      Q, Q_BAR: INOUT STD_LOGIC);
   END COMPONENT;
   SIGNAL u,v: STD_LOGIC;
   SIGNAL NOTC: STD_LOGIC := NOT C;
BEGIN
   u1: d_latch port map (D, C, u, v);
   u2: d_latch port map (D, NOTC, Q, Q_BAR);
END behavioral;
