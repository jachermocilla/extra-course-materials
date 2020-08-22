LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY mux_4bit IS
   PORT (A, B: IN STD_LOGIC_VECTOR(3 DOWNTO 0);
      S: IN STD_LOGIC;
      C: OUT STD_LOGIC_VECTOR(3 DOWNTO 0));
END mux_4bit;
--------------------------------
ARCHITECTURE behavioral OF mux_4bit IS
   COMPONENT mux_2to1 IS
      PORT (A, B, S: IN STD_LOGIC;
      C: OUT STD_LOGIC);
   END COMPONENT;
BEGIN
   Q1: FOR I IN 0 TO 3 GENERATE
      u1: mux_2to1 port map(A(I), B(I), S, C(I));
   END GENERATE;
END behavioral;
