LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY and_or IS
   PORT (a, b, op: IN STD_LOGIC;
   result: OUT STD_LOGIC);
END and_or;
--------------------------------

ARCHITECTURE behavioral OF and_or IS

   COMPONENT mux_2to1 IS
      PORT (A, B, S: IN STD_LOGIC;
      C: OUT STD_LOGIC);
   END COMPONENT;

   COMPONENT and_gate IS
      PORT (A, B: IN STD_LOGIC;
      C: OUT STD_LOGIC);
   END COMPONENT;

   COMPONENT or_gate IS
      PORT (A, B: IN STD_LOGIC;
      C: OUT STD_LOGIC);
   END COMPONENT;

   signal and_out: STD_LOGIC;
   signal or_out: STD_LOGIC;

BEGIN

   u1: and_gate port map(a, b, and_out);
   u2: or_gate port map(a, b, or_out);
   u3: mux_2to1 port map(and_out, or_out, op); 

END behavioral;
