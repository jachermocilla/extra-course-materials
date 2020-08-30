LIBRARY ieee;
USE ieee.std_logic_1164.all;
ENTITY reg_file2_tb IS
END reg_file2_tb;
ARCHITECTURE behavior OF reg_file2_tb IS
   CONSTANT frequency: integer := 100e6; 
   CONSTANT period : time := 1000 ms / frequency;
   SIGNAL clk : std_logic := '0';

   COMPONENT reg_file2 is
      PORT( 
         read_n1: IN STD_LOGIC;
         read_n2: IN STD_LOGIC;
         write_n: IN STD_LOGIC;
         write_data: IN STD_LOGIC;
         write: IN STD_LOGIC;
         clk: IN STD_LOGIC;
         read_data1: OUT STD_LOGIC;
         read_data2: OUT STD_LOGIC);
   END COMPONENT;

   SIGNAL inputs: STD_LOGIC_VECTOR(5 DOWNTO 0);
   SIGNAL outputs: STD_LOGIC_VECTOR(1 DOWNTO 0);
BEGIN 
   clk <= not clk after period;
   uut: reg_file2 PORT MAP (read_n1=>inputs(5),read_n2=>inputs(4),write_n=>inputs(3),write_data=>inputs(2),write=>inputs(1),clk=>clk,read_data1=>outputs(1),read_data2=>outputs(0)); 
   stim_proc: PROCESS
   BEGIN
      inputs <= "000000";
      WAIT FOR period / 2 ;
      -- write 1 to register 0
      inputs <= "000110";
      WAIT FOR 80 ns;

      inputs <= "000010";
      WAIT FOR 80 ns;
--      D <= '1';
      WAIT;
   END PROCESS;
END ARCHITECTURE;
