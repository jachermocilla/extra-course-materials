LIBRARY ieee;
USE ieee.std_logic_1164.all;
ENTITY trafficlite_tb IS
END trafficlite_tb;
ARCHITECTURE behavior OF trafficlite_tb IS
   --100Mhz
   CONSTANT frequency: integer := 100e6; 
   CONSTANT period : time := 1000 ms / frequency;
   COMPONENT trafficlite IS
   PORT (EWCar, NSCar, clk: IN STD_LOGIC;
      EWLite, NSLite: OUT STD_LOGIC
   );
   END COMPONENT;
   SIGNAL clk : STD_LOGIC := '0';
   SIGNAL inputs : STD_LOGIC_VECTOR(1 DOWNTO 0);
   SIGNAL outputs : STD_LOGIC_VECTOR(1 DOWNTO 0);
BEGIN 
   clk <= NOT clk AFTER period / 2;
   uut: trafficlite PORT MAP (inputs(1),inputs(0),clk,outputs(1),outputs(0));
   stim_proc: PROCESS
   BEGIN
      inputs <= "00";
      WAIT FOR period ;
      inputs <= "10";
      WAIT FOR period ;
      inputs <= "01";
      WAIT FOR period ;
   END PROCESS;
END ARCHITECTURE;
