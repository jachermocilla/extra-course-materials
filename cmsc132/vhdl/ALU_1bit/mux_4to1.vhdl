LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY mux_4to1 IS
   PORT (A, B, C, D, S0, S1: IN STD_LOGIC;
   E: OUT STD_LOGIC);
END mux_4to1;
--------------------------------
ARCHITECTURE behavioral OF mux_4to1 IS
   COMPONENT mux_2to1 IS
      PORT (A, B, S: IN STD_LOGIC;
      C: OUT STD_LOGIC);
   END COMPONENT;
   SIGNAL u: STD_LOGIC;
   SIGNAL v: STD_LOGIC;
BEGIN
   u1: mux_2to1 port map(A, B, S1, u);
   u2: mux_2to1 port map(C, D, S1, v);
   u3: mux_2to1 port map(u, v, S0, E); 
END behavioral;
