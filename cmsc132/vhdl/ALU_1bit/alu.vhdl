LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY alu IS
   PORT (a, b, CarryIn : IN STD_LOGIC;
   Operation: IN STD_LOGIC_VECTOR(1 DOWNTO 0);
   Result, CarryOut: OUT STD_LOGIC);
END alu;
--------------------------------
ARCHITECTURE behavioral OF alu IS
   COMPONENT mux_4to1 IS
      PORT (A, B, C, D, S0, S1: IN STD_LOGIC;
      E: OUT STD_LOGIC);
   END COMPONENT;
   COMPONENT and_gate IS
      PORT (A, B: IN STD_LOGIC; C: OUT STD_LOGIC);
   END COMPONENT;
   COMPONENT or_gate IS
      PORT (A, B: IN STD_LOGIC; C: OUT STD_LOGIC);
   END COMPONENT;
   COMPONENT full_adder is
      PORT ( a : IN STD_LOGIC; b : IN STD_LOGIC;
         CarryIn : IN STD_LOGIC;Sum : OUT STD_LOGIC;
         CarryOut : OUT STD_LOGIC);
   END COMPONENT;
   SIGNAL u, v, w, x: STD_LOGIC;
BEGIN
   u1: and_gate port map(a, b, u);
   u2: or_gate port map(a, b, v);
   u3: full_adder port map(a, b, CarryIn, w, CarryOut); 
   u4: mux_4to1 port map(u, v, w, x, Operation(0), Operation(1), Result);
END behavioral;
