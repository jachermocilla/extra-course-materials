LIBRARY ieee;
USE ieee.std_logic_1164.all;

ENTITY mux_4to1_tb IS
END mux_4to1_tb;

ARCHITECTURE behavior OF mux_4to1_tb IS
   COMPONENT mux_4to1 IS 
      PORT (
         A, B, C, D, S0, S1: IN STD_LOGIC;
         E: OUT STD_LOGIC);
   END COMPONENT;
   SIGNAL input: STD_LOGIC_VECTOR(5 DOWNTO 0);
   SIGNAL output: STD_LOGIC;

BEGIN 
   uut: mux_4to1 PORT MAP (
      S0 => input(5),
      S1 => input(4),
      A => input(3),
      B => input(2),
      C => input(1),
      D => input(0),
      E => output
   );

   stim_proc: PROCESS
   BEGIN
      input <= "001011"; wait for 10 ns; assert output='1' report "00 failed,output= " & std_logic'image(output);
      input <= "011011"; wait for 10 ns; assert output='0' report "01 failed,output= " & std_logic'image(output);
      input <= "101011"; wait for 10 ns; assert output='1' report "10 failed,output= " & std_logic'image(output);
      input <= "111011"; wait for 10 ns; assert output='0' report "11 failed,output= " & std_logic'image(output);
      WAIT;
   END PROCESS;
END;      
   

