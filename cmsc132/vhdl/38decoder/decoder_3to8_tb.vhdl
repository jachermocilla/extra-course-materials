LIBRARY ieee;
USE ieee.std_logic_1164.all;

ENTITY decoder_3to8_tb IS
END decoder_3to8_tb;

architecture behavior of decoder_3to8_tb is
   component decoder_3to8 is 
      port (
         INPUT: IN STD_LOGIC_VECTOR(2 downto 0);
         OUTPUT: OUT STD_LOGIC_VECTOR(7 downto 0));
   end component;
   signal INPUT_TB: STD_LOGIC_VECTOR(2 downto 0);
   signal OUTPUT_TB: STD_LOGIC_VECTOR(7 downto 0);

begin 
   uut: decoder_3to8 port map (
      INPUT => INPUT_TB,
      OUTPUT => OUTPUT_TB
   );

   stim_proc: process
   begin

      wait for 50 ns;


      INPUT_TB <= "000"; wait for 10 ns; assert OUTPUT_TB="00000000" report "000 failed,output= ";
      INPUT_TB <= "001"; wait for 10 ns; assert OUTPUT_TB="00000001" report "001 failed,output= ";
      INPUT_TB <= "010"; wait for 10 ns; assert OUTPUT_TB="00000100" report "010 failed,output= "; 
      INPUT_TB <= "011"; wait for 10 ns; assert OUTPUT_TB="00001000" report "011 failed,output= ";
      INPUT_TB <= "100"; wait for 10 ns; assert OUTPUT_TB="00010000" report "100 failed,output= ";
      INPUT_TB <= "101"; wait for 10 ns; assert OUTPUT_TB="00100000" report "101 failed,output= "; 
      INPUT_TB <= "110"; wait for 10 ns; assert OUTPUT_TB="01000000" report "110 failed,output= ";
      INPUT_TB <= "111"; wait for 10 ns; assert OUTPUT_TB="10000000" report "111 failed,output= ";
      wait;
   end process;
end;      
   

