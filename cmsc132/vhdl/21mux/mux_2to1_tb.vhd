LIBRARY ieee;
USE ieee.std_logic_1164.all;

ENTITY mux_2to1_tb IS
END mux_2to1_tb;

architecture behavior of mux_2to1_tb is
   component mux_2to1 is 
      port (
         A, B, S: IN STD_LOGIC;
         C: OUT STD_LOGIC);
   end component;
   signal input: STD_LOGIC_VECTOR(2 downto 0);
   signal output: STD_LOGIC;

begin 
   uut: mux_2to1 port map (
      A => input(1),
      B => input(2),
      S => input(0),
      C => output
   );

   stim_proc: process
   begin
      input <= "000"; wait for 10 ns; assert output='0' report "000 failed,output= " & std_logic'image(output);
      input <= "001"; wait for 10 ns; assert output='0' report "001 failed,output= " & std_logic'image(output);
      input <= "010"; wait for 10 ns; assert output='1' report "010 failed,output= " & std_logic'image(output); 
      input <= "011"; wait for 10 ns; assert output='1' report "011 failed,output= " & std_logic'image(output);
      input <= "100"; wait for 10 ns; assert output='0' report "100 failed,output= " & std_logic'image(output);
      input <= "101"; wait for 10 ns; assert output='1' report "101 failed,output= " & std_logic'image(output); 
      input <= "110"; wait for 10 ns; assert output='0' report "110 failed,output= " & std_logic'image(output);
      input <= "111"; wait for 10 ns; assert output='1' report "111 failed,output= " & std_logic'image(output);
      wait;
   end process;
end;      
   

