LIBRARY ieee;
USE ieee.std_logic_1164.all;
ENTITY and_or_tb IS
END and_or_tb;
ARCHITECTURE behavior OF and_or_tb IS
   COMPONENT and_or IS 
      PORT (
         a, b, op: IN STD_LOGIC;
         result: OUT STD_LOGIC);
   END COMPONENT;

   SIGNAL input: STD_LOGIC_VECTOR(2 DOWNTO 0);
   SIGNAL output: STD_LOGIC;

BEGIN 
   UUT: and_or PORT MAP (
      a => input(1),
      b => input(2),
      op => input(0),
      result => output
   );

   STIM_PROC: PROCESS
   BEGIN
      input <= "000"; WAIT FOR 10 NS; ASSERT output='0' REPORT "000 failed,output= " & STD_LOGIC'IMAGE(output);
      input <= "001"; WAIT FOR 10 NS; ASSERT output='0' REPORT "001 failed,output= " & STD_LOGIC'IMAGE(output);
      input <= "010"; WAIT FOR 10 NS; ASSERT output='0' REPORT "011 failed,output= " & STD_LOGIC'IMAGE(output);
      input <= "011"; WAIT FOR 10 NS; ASSERT output='1' REPORT "011 failed,output= " & STD_LOGIC'IMAGE(output);
      input <= "100"; WAIT FOR 10 NS; ASSERT output='0' REPORT "100 failed,output= " & STD_LOGIC'IMAGE(output);
      input <= "101"; WAIT FOR 10 NS; ASSERT output='1' REPORT "100 failed,output= " & STD_LOGIC'IMAGE(output);
      input <= "110"; WAIT FOR 10 NS; ASSERT output='1' REPORT "110 failed,output= " & STD_LOGIC'IMAGE(output);
      input <= "111"; WAIT FOR 10 NS; ASSERT output='1' REPORT "111 failed,output= " & STD_LOGIC'IMAGE(output);
      WAIT;
   END PROCESS;
END;      
   

