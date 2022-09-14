LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY and_or IS
   PORT (a, b, Operation: IN STD_LOGIC;
   Result: OUT STD_LOGIC);
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
   SIGNAL u: STD_LOGIC;
   SIGNAL v: STD_LOGIC;
BEGIN
   u1_and: and_gate port map(a, b, u);
   u2_or: or_gate port map(a, b, v);
   u3_mux: mux_2to1 port map(u, v,
                   Operation,
                   Result); 
END behavioral;
