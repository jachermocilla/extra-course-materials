LIBRARY ieee;
USE ieee.std_logic_1164.all;
ENTITY dff_tb IS
END dff_tb;
ARCHITECTURE behavior OF dff_tb IS
   CONSTANT frequency: integer := 100e6; 
   CONSTANT period : time := 1000 ms / frequency;
   SIGNAL clk : std_logic := '0';
   COMPONENT dff is
   PORT( D: IN STD_LOGIC;
      C: IN STD_LOGIC;
      Q: INOUT STD_LOGIC;
      Q_BAR: INOUT STD_LOGIC);
   END COMPONENT;
   SIGNAL D: STD_LOGIC := '0';
   SIGNAL Q: STD_LOGIC := '0';
   SIGNAL Q_BAR: STD_LOGIC;
BEGIN 
   clk <= not clk after period;
   uut: dff PORT MAP (D, clk, Q, Q_BAR);
   stim_proc: PROCESS
   BEGIN
      D <= '0';
      WAIT FOR period / 2 ;
      D <= '1';
      WAIT FOR 20 ns;
      D <= '0';
      WAIT FOR 20 ns;
      D <= '1';
      WAIT;
   END PROCESS;
END ARCHITECTURE;
