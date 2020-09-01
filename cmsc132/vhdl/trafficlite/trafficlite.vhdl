LIBRARY ieee;
USE ieee.std_logic_1164.all;
---------------------------------
ENTITY trafficlight IS
   PORT (
      EWCar, NSCar, clk: IN STD_LOGIC;
      EWLite, NSLite: OUT STD_LOGIC
   );
END trafficlight;
--------------------------------
ARCHITECTURE pure_logic OF trafficlight IS
   SIGNAL state : STD_LOGIC := '0'; 
BEGIN
   PROCESS
   BEGIN
      NSLite <= NOT state;
      EWLite <= state;
      IF (RISING_EDGE(clk)) THEN
         CASE state IS
            WHEN '0' =>
               state <= EWCar;
            WHEN '1' =>
               state <= NSCar;
            WHEN others => 
         END CASE;
      END IF; 
   END PROCESS;
END pure_logic;
