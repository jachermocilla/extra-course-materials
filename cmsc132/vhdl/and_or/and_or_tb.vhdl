LIBRARY ieee;
USE ieee.std_logic_1164.all;

ENTITY and_or_tb IS
END and_or_tb;

architecture behavior of and_or_tb is
   component and_or is 
      port (
         a, b, op: IN STD_LOGIC;
         result: OUT STD_LOGIC);
   end component;

   signal input: STD_LOGIC_VECTOR(2 downto 0);
   signal output: STD_LOGIC;

begin 
   uut: and_or port map (
      a => input(0),
      b => input(1),
      op => input(0),
      result => output
   );

   stim_proc: process
   begin
      input <= "000"; wait for 10 ns; assert output='0' report "000 failed,output= " & std_logic'image(output);
      input <= "001"; wait for 10 ns; assert output='0' report "001 failed,output= " & std_logic'image(output);
      input <= "010"; wait for 10 ns; assert output='0' report "011 failed,output= " & std_logic'image(output);
      input <= "011"; wait for 10 ns; assert output='1' report "011 failed,output= " & std_logic'image(output);
      input <= "100"; wait for 10 ns; assert output='0' report "100 failed,output= " & std_logic'image(output);
      input <= "101"; wait for 10 ns; assert output='1' report "100 failed,output= " & std_logic'image(output);
      input <= "110"; wait for 10 ns; assert output='1' report "110 failed,output= " & std_logic'image(output);
      input <= "111"; wait for 10 ns; assert output='1' report "111 failed,output= " & std_logic'image(output);
      wait;
   end process;
end;      
   

