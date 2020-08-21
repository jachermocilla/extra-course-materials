LIBRARY ieee;
USE ieee.std_logic_1164.all;

ENTITY mux_2to1_tb IS
END mux_2to1_tb;

ARCHITECTURE behavior OF mux_2to1_tb IS
   COMPONENT mux_2to1 IS 
      PORT (
         A, B, S: IN STD_LOGIC;
         C: OUT STD_LOGIC);
   END COMPONENT;
   SIGNAL input: STD_LOGIC_VECTOR(2 DOWNTO 0);
   SIGNAL output: STD_LOGIC;

BEGIN 
   uut: mux_2to1 PORT MAP (
      S => input(0),
      A => input(1),
      B => input(2),
      C => output
   );

   stim_proc: PROCESS
   BEGIN
      input <= "010"; wait for 10 ns; assert output='1' report "000 failed,output= " & std_logic'image(output);
      input <= "101"; wait for 10 ns; assert output='1' report "001 failed,output= " & std_logic'image(output);
      input <= "000"; wait for 10 ns; assert output='0' report "000 failed,output= " & std_logic'image(output);
      input <= "100"; wait for 10 ns; assert output='0' report "001 failed,output= " & std_logic'image(output);
      WAIT;
   END PROCESS;
END;      
   

